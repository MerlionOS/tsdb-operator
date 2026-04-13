package backup

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r), nil }

type fakeUploader struct {
	mu     sync.Mutex
	puts   []*s3.PutObjectInput
	bodies [][]byte
}

// PutObject drains the body before returning so streaming-pipe callers see
// the same flow they would with real S3.
func (f *fakeUploader) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	body, _ := io.ReadAll(in.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.puts = append(f.puts, in)
	f.bodies = append(f.bodies, body)
	return &s3.PutObjectOutput{}, nil
}

// StreamUpload mirrors PutObject for the multipart code path.
func (f *fakeUploader) StreamUpload(_ context.Context, in *s3.PutObjectInput) error {
	body, _ := io.ReadAll(in.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.puts = append(f.puts, in)
	f.bodies = append(f.bodies, body)
	return nil
}

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := observabilityv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRunOnceUploadsAndStampsStatus(t *testing.T) {
	s := newScheme(t)
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
		Spec: observabilityv1.PrometheusClusterSpec{
			Backup: observabilityv1.S3BackupSpec{
				Enabled: true, Bucket: "b", Prefix: "p",
			},
		},
	}
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(pc).
		WithStatusSubresource(&observabilityv1.PrometheusCluster{}).
		Build()

	snapshotBody := "FAKE_SNAPSHOT"
	rt := roundTripFunc(func(r *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(snapshotBody)),
			Header:     make(http.Header),
		}
	})
	up := &fakeUploader{}
	sched := &Scheduler{
		Client: c,
		S3:     up,
		HTTP:   &http.Client{Transport: rt},
	}

	if err := sched.RunOnce(context.Background(), "ns", "demo"); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if len(up.puts) != 1 {
		t.Fatalf("expected 1 PutObject, got %d", len(up.puts))
	}
	got := up.puts[0]
	if *got.Bucket != "b" {
		t.Errorf("bucket = %q", *got.Bucket)
	}
	if !strings.HasPrefix(*got.Key, "p/demo/") || !strings.HasSuffix(*got.Key, ".tar") {
		t.Errorf("unexpected key: %q", *got.Key)
	}

	var updated observabilityv1.PrometheusCluster
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "ns", Name: "demo"}, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.LastBackupTime == nil {
		t.Fatal("LastBackupTime should be set")
	}
}

type fakeExec struct {
	commands [][]string
	stdout   []byte
}

func (f *fakeExec) Exec(_ context.Context, _, _, _ string, cmd []string, w io.Writer) error {
	f.commands = append(f.commands, cmd)
	if len(cmd) > 0 && cmd[0] == "tar" {
		_, _ = w.Write(f.stdout)
	}
	return nil
}

func TestRunOnceTarStreamsToS3WhenExecConfigured(t *testing.T) {
	s := newScheme(t)
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
		Spec: observabilityv1.PrometheusClusterSpec{
			Backup: observabilityv1.S3BackupSpec{Enabled: true, Bucket: "b", Prefix: "p"},
		},
	}
	c := fake.NewClientBuilder().
		WithScheme(s).WithObjects(pc).
		WithStatusSubresource(&observabilityv1.PrometheusCluster{}).Build()

	rt := roundTripFunc(func(_ *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(
				`{"status":"success","data":{"name":"20260413T000000Z-abc"}}`)),
			Header: make(http.Header),
		}
	})
	up := &fakeUploader{}
	exec := &fakeExec{stdout: []byte("FAKE_TAR_BYTES")}
	sched := &Scheduler{
		Client: c, S3: up, Exec: exec,
		HTTP: &http.Client{Transport: rt},
	}
	if err := sched.RunOnce(context.Background(), "ns", "demo"); err != nil {
		t.Fatal(err)
	}

	if len(up.puts) != 1 {
		t.Fatalf("want 1 PutObject, got %d", len(up.puts))
	}
	if string(up.bodies[0]) != "FAKE_TAR_BYTES" {
		t.Errorf("uploaded body = %q, want FAKE_TAR_BYTES", up.bodies[0])
	}

	// Two execs: tar then cleanup.
	if len(exec.commands) != 2 {
		t.Fatalf("want 2 exec commands, got %d", len(exec.commands))
	}
	if exec.commands[0][0] != "tar" {
		t.Errorf("first exec = %v, want tar", exec.commands[0])
	}
	if exec.commands[1][0] != "rm" {
		t.Errorf("second exec = %v, want rm", exec.commands[1])
	}
}

func TestRunOnceMissingClusterErrors(t *testing.T) {
	s := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(s).Build()
	sched := &Scheduler{Client: c, S3: &fakeUploader{}, HTTP: &http.Client{}}
	if err := sched.RunOnce(context.Background(), "ns", "nope"); err == nil {
		t.Fatal("expected error for missing cluster")
	}
}
