package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

func init() { gin.SetMode(gin.TestMode) }

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

func newServer(t *testing.T, objs ...client.Object) *Server {
	t.Helper()
	s := newScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objs...).
		WithStatusSubresource(&observabilityv1.PrometheusCluster{}).
		Build()
	return &Server{Client: c, Namespace: "ns"}
}

func do(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	return rec
}

func TestListClusters(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "ns"},
	}
	srv := newServer(t, pc)
	rec := do(t, srv, http.MethodGet, "/api/clusters", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got []observabilityv1.PrometheusCluster
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "a" {
		t.Fatalf("unexpected list: %+v", got)
	}
}

func TestCreateCluster(t *testing.T) {
	srv := newServer(t)
	body := map[string]any{
		"name": "demo",
		"spec": map[string]any{"replicas": 2},
	}
	rec := do(t, srv, http.MethodPost, "/api/clusters", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var created observabilityv1.PrometheusCluster
	if err := srv.Client.Get(t.Context(), client.ObjectKey{Namespace: "ns", Name: "demo"}, &created); err != nil {
		t.Fatalf("not persisted: %v", err)
	}
	if created.Spec.Replicas != 2 {
		t.Errorf("replicas = %d", created.Spec.Replicas)
	}
}

func TestCreateClusterBadJSON(t *testing.T) {
	srv := newServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetClusterNotFound(t *testing.T) {
	srv := newServer(t)
	rec := do(t, srv, http.MethodGet, "/api/clusters/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetClusterOK(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	srv := newServer(t, pc)
	rec := do(t, srv, http.MethodGet, "/api/clusters/demo", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteCluster(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	srv := newServer(t, pc)
	rec := do(t, srv, http.MethodDelete, "/api/clusters/demo", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	err := srv.Client.Get(t.Context(), client.ObjectKey{Namespace: "ns", Name: "demo"}, &observabilityv1.PrometheusCluster{})
	if err == nil {
		t.Fatal("expected cluster to be deleted")
	}
}

func TestBackupWithoutScheduler(t *testing.T) {
	srv := newServer(t)
	rec := do(t, srv, http.MethodPost, "/api/clusters/demo/backup", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestListClusterSets(t *testing.T) {
	set := &observabilityv1.PrometheusClusterSet{
		ObjectMeta: metav1.ObjectMeta{Name: "global"},
	}
	srv := newServer(t, set)
	rec := do(t, srv, http.MethodGet, "/api/clustersets", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got []observabilityv1.PrometheusClusterSet
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "global" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetClusterSetNotFound(t *testing.T) {
	srv := newServer(t)
	rec := do(t, srv, http.MethodGet, "/api/clustersets/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAuditWithoutLogger(t *testing.T) {
	srv := newServer(t)
	rec := do(t, srv, http.MethodGet, "/api/clusters/demo/audit", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
