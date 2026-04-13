// Package backup performs scheduled TSDB snapshots and uploads them to
// S3-compatible object storage (AWS S3, MinIO, etc.).
package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
	"github.com/MerlionOS/tsdb-operator/internal/metrics"
)

// Uploader abstracts the S3 surface we need. Two methods because PutObject
// requires a seekable Body (real S3 SDK rejects piped readers over plain
// HTTP), while StreamUpload uses S3 multipart and works with any io.Reader.
type Uploader interface {
	PutObject(ctx context.Context, in *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	StreamUpload(ctx context.Context, in *s3.PutObjectInput) error
}

// PodExecutor runs a command in a pod's container and streams stdout back.
// Separated so tests can inject fakes without a real Kubernetes cluster.
type PodExecutor interface {
	Exec(ctx context.Context, namespace, pod, container string, cmd []string, stdout io.Writer) error
}

// Scheduler runs backups on a cron schedule per PrometheusCluster.
type Scheduler struct {
	Client  client.Client
	S3      Uploader
	Exec    PodExecutor // nil falls back to snapshot-API-only mode
	HTTP    *http.Client
	cron    *cron.Cron
	entries map[string]cron.EntryID
}

// New creates a Scheduler ready for Start.
func New(c client.Client, s3c Uploader) *Scheduler {
	return &Scheduler{
		Client:  c,
		S3:      s3c,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		cron:    cron.New(),
		entries: map[string]cron.EntryID{},
	}
}

// Start begins the scheduler; stops when ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	s.cron.Start()
	<-ctx.Done()
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()
	return nil
}

// Register adds or replaces the schedule entry for a cluster.
func (s *Scheduler) Register(ctx context.Context, pc *observabilityv1.PrometheusCluster) error {
	if !pc.Spec.Backup.Enabled || pc.Spec.Backup.Schedule == "" {
		return nil
	}
	key := pc.Namespace + "/" + pc.Name
	if id, ok := s.entries[key]; ok {
		s.cron.Remove(id)
	}
	name, ns := pc.Name, pc.Namespace
	id, err := s.cron.AddFunc(pc.Spec.Backup.Schedule, func() {
		if err := s.RunOnce(context.Background(), ns, name); err != nil {
			logf.Log.WithName("backup").Error(err, "scheduled backup failed", "cluster", key)
		}
	})
	if err != nil {
		return fmt.Errorf("add cron entry: %w", err)
	}
	s.entries[key] = id
	return nil
}

// RunOnce triggers a snapshot + upload for the named cluster. When a
// PodExecutor is configured, the on-disk snapshot directory is tar-streamed
// into S3. Otherwise the (legacy) raw admin-API response is uploaded — used
// only in tests that can't exec into a pod.
func (s *Scheduler) RunOnce(ctx context.Context, namespace, name string) (err error) {
	defer func() {
		result := "success"
		if err != nil {
			result = "error"
		}
		metrics.BackupTotal.WithLabelValues(namespace, name, result).Inc()
	}()

	var pc observabilityv1.PrometheusCluster
	if err := s.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &pc); err != nil {
		return fmt.Errorf("get cluster: %w", err)
	}

	snapName, rawResp, err := s.snapshot(ctx, &pc)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	key := fmt.Sprintf("%s/%s/%s.tar", pc.Spec.Backup.Prefix, pc.Name, time.Now().UTC().Format("20060102T150405Z"))

	if s.Exec == nil || snapName == "" {
		// Fallback: upload the admin-API response body. Not a usable
		// restore artifact — see docs/RESTORE.md.
		if _, err := s.S3.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(pc.Spec.Backup.Bucket),
			Key:    aws.String(key),
			Body:   rawResp,
		}); err != nil {
			return fmt.Errorf("put object (fallback): %w", err)
		}
	} else {
		if err := s.tarStreamToS3(ctx, &pc, snapName, key); err != nil {
			return fmt.Errorf("tar stream: %w", err)
		}
		// Best-effort cleanup on the pod so snapshot dirs don't accumulate.
		if cleanupErr := s.cleanupSnapshot(ctx, &pc, snapName); cleanupErr != nil {
			logf.FromContext(ctx).Error(cleanupErr, "snapshot cleanup failed",
				"cluster", pc.Name, "snapshot", snapName)
		}
	}

	now := metav1.Now()
	pc.Status.LastBackupTime = &now
	if err := s.Client.Status().Update(ctx, &pc); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// snapshotResp mirrors Prometheus' admin snapshot API response shape.
type snapshotResp struct {
	Status string `json:"status"`
	Data   struct {
		Name string `json:"name"`
	} `json:"data"`
}

// snapshot POSTs to the admin snapshot endpoint and returns the snapshot
// directory name plus the raw bytes (the latter is the fallback artifact).
func (s *Scheduler) snapshot(ctx context.Context, pc *observabilityv1.PrometheusCluster) (string, io.Reader, error) {
	url := fmt.Sprintf("http://%s.%s.svc:9090/api/v1/admin/tsdb/snapshot", pc.Name, pc.Namespace)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("trigger snapshot: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read snapshot response: %w", err)
	}
	var parsed snapshotResp
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Status == "success" {
		return parsed.Data.Name, bytesReader(body), nil
	}
	return "", bytesReader(body), nil
}

// tarStreamToS3 execs `tar -C /prometheus/snapshots -cf - <name>` inside the
// Prometheus container and streams the stdout pipe into S3 PutObject.
func (s *Scheduler) tarStreamToS3(ctx context.Context, pc *observabilityv1.PrometheusCluster, snapName, key string) error {
	pod := pc.Name + "-0"
	pr, pw := io.Pipe()
	execErr := make(chan error, 1)
	go func() {
		err := s.Exec.Exec(ctx, pc.Namespace, pod, "prometheus",
			[]string{"tar", "-C", "/prometheus/snapshots", "-cf", "-", snapName}, pw)
		_ = pw.CloseWithError(err)
		execErr <- err
	}()

	putErr := s.S3.StreamUpload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(pc.Spec.Backup.Bucket),
		Key:    aws.String(key),
		Body:   pr,
	})
	_ = pr.Close()
	if err := <-execErr; err != nil {
		return fmt.Errorf("exec tar: %w", err)
	}
	if putErr != nil {
		return fmt.Errorf("put object: %w", putErr)
	}
	return nil
}

// cleanupSnapshot removes the on-disk snapshot directory after a successful
// upload. Failures are logged but not propagated — they'd be reported as
// backup failures on the next run anyway if they accumulate.
func (s *Scheduler) cleanupSnapshot(ctx context.Context, pc *observabilityv1.PrometheusCluster, snapName string) error {
	pod := pc.Name + "-0"
	return s.Exec.Exec(ctx, pc.Namespace, pod, "prometheus",
		[]string{"rm", "-rf", "/prometheus/snapshots/" + snapName}, io.Discard)
}

// bytesReader wraps a byte slice without importing bytes in every caller.
func bytesReader(b []byte) io.Reader { return &byteReader{b: b} }

type byteReader struct {
	b []byte
	i int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
