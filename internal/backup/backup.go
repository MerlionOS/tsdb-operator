// Package backup performs scheduled TSDB snapshots and uploads them to
// S3-compatible object storage (AWS S3, MinIO, etc.).
package backup

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

// Uploader abstracts the S3 PutObject surface we need (swappable in tests).
type Uploader interface {
	PutObject(ctx context.Context, in *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// Scheduler runs backups on a cron schedule per PrometheusCluster.
type Scheduler struct {
	Client  client.Client
	S3      Uploader
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

// RunOnce triggers a snapshot + upload for the named cluster.
func (s *Scheduler) RunOnce(ctx context.Context, namespace, name string) error {
	var pc observabilityv1.PrometheusCluster
	if err := s.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &pc); err != nil {
		return fmt.Errorf("get cluster: %w", err)
	}
	data, err := s.snapshot(ctx, &pc)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	key := fmt.Sprintf("%s/%s/%s.tar", pc.Spec.Backup.Prefix, pc.Name, time.Now().UTC().Format("20060102T150405Z"))
	_, err = s.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(pc.Spec.Backup.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	now := metav1.Now()
	pc.Status.LastBackupTime = &now
	if err := s.Client.Status().Update(ctx, &pc); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// snapshot triggers Prometheus /api/v1/admin/tsdb/snapshot and collects payload.
// Real implementation would tar the snapshot directory off the pod filesystem;
// we return a placeholder marker to keep the control path testable.
func (s *Scheduler) snapshot(ctx context.Context, pc *observabilityv1.PrometheusCluster) ([]byte, error) {
	url := fmt.Sprintf("http://%s.%s.svc:9090/api/v1/admin/tsdb/snapshot", pc.Name, pc.Namespace)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trigger snapshot: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("read snapshot response: %w", err)
	}
	return buf.Bytes(), nil
}
