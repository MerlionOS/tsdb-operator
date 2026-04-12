// Package api exposes a gin HTTP server for managing PrometheusCluster
// resources, triggering manual backups, and querying the audit log.
package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
	"github.com/MerlionOS/tsdb-operator/internal/audit"
	"github.com/MerlionOS/tsdb-operator/internal/backup"
)

// Server wires the HTTP router to the Kubernetes client and helpers.
type Server struct {
	Client    client.Client
	Namespace string
	Audit     *audit.Logger
	Backup    *backup.Scheduler
}

// Router builds a gin.Engine with all routes registered.
func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api")
	api.GET("/clusters", s.listClusters)
	api.POST("/clusters", s.createCluster)
	api.GET("/clusters/:name", s.getCluster)
	api.DELETE("/clusters/:name", s.deleteCluster)
	api.POST("/clusters/:name/backup", s.triggerBackup)
	api.GET("/clusters/:name/audit", s.queryAudit)
	return r
}

func (s *Server) listClusters(c *gin.Context) {
	var list observabilityv1.PrometheusClusterList
	if err := s.Client.List(c.Request.Context(), &list, client.InNamespace(s.Namespace)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list.Items)
}

type createReq struct {
	Name string                                `json:"name" binding:"required"`
	Spec observabilityv1.PrometheusClusterSpec `json:"spec"`
}

func (s *Server) createCluster(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: s.Namespace},
		Spec:       req.Spec,
	}
	if err := s.Client.Create(c.Request.Context(), pc); err != nil {
		c.JSON(statusFor(err), gin.H{"error": err.Error()})
		return
	}
	s.audit(c, req.Name, "create", "success", "")
	c.JSON(http.StatusCreated, pc)
}

func (s *Server) getCluster(c *gin.Context) {
	name := c.Param("name")
	var pc observabilityv1.PrometheusCluster
	if err := s.Client.Get(c.Request.Context(), client.ObjectKey{Namespace: s.Namespace, Name: name}, &pc); err != nil {
		c.JSON(statusFor(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pc)
}

func (s *Server) deleteCluster(c *gin.Context) {
	name := c.Param("name")
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.Namespace},
	}
	if err := s.Client.Delete(c.Request.Context(), pc); err != nil {
		s.audit(c, name, "delete", "error", err.Error())
		c.JSON(statusFor(err), gin.H{"error": err.Error()})
		return
	}
	s.audit(c, name, "delete", "success", "")
	c.Status(http.StatusNoContent)
}

func (s *Server) triggerBackup(c *gin.Context) {
	name := c.Param("name")
	if s.Backup == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backup scheduler not configured"})
		return
	}
	if err := s.Backup.RunOnce(c.Request.Context(), s.Namespace, name); err != nil {
		s.audit(c, name, "backup", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, name, "backup", "success", "")
	c.JSON(http.StatusAccepted, gin.H{"status": "ok"})
}

func (s *Server) queryAudit(c *gin.Context) {
	name := c.Param("name")
	if s.Audit == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audit log not configured"})
		return
	}
	entries, err := s.Audit.Query(c.Request.Context(), name, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (s *Server) audit(c *gin.Context, cluster, op, result, detail string) {
	if s.Audit == nil {
		return
	}
	operator := c.GetHeader("X-Operator")
	if operator == "" {
		operator = "anonymous"
	}
	_ = s.Audit.Record(c.Request.Context(), audit.Entry{
		ClusterName: cluster,
		Operation:   op,
		Operator:    operator,
		Result:      result,
		Detail:      detail,
	})
}

func statusFor(err error) int {
	switch {
	case apierrors.IsNotFound(err):
		return http.StatusNotFound
	case apierrors.IsAlreadyExists(err):
		return http.StatusConflict
	case apierrors.IsInvalid(err):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// ListenAndServe runs the HTTP server until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	srv := &http.Server{Addr: addr, Handler: s.Router()}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	}
}
