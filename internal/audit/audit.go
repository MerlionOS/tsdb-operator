// Package audit persists cluster-management operations to a PostgreSQL table.
package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/MerlionOS/tsdb-operator/internal/metrics"
)

// Entry is one row in the audit log.
type Entry struct {
	ID          int64     `db:"id" json:"id"`
	ClusterName string    `db:"cluster_name" json:"clusterName"`
	Operation   string    `db:"operation" json:"operation"`
	Operator    string    `db:"operator" json:"operator"`
	Timestamp   time.Time `db:"timestamp" json:"timestamp"`
	Result      string    `db:"result" json:"result"`
	Detail      string    `db:"detail" json:"detail"`
}

// Logger writes audit entries to Postgres.
type Logger struct {
	db            *sqlx.DB
	RetentionDays int
	PruneInterval time.Duration
}

const schema = `
CREATE TABLE IF NOT EXISTS audit_log (
    id            BIGSERIAL PRIMARY KEY,
    cluster_name  TEXT        NOT NULL,
    operation     TEXT        NOT NULL,
    operator      TEXT        NOT NULL,
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result        TEXT        NOT NULL,
    detail        TEXT        NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS audit_log_cluster_ts ON audit_log(cluster_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS audit_log_ts ON audit_log(timestamp);
`

// Open dials Postgres and ensures the schema exists.
func Open(ctx context.Context, dsn string) (*Logger, error) {
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}
	return &Logger{
		db:            db,
		RetentionDays: 0,             // 0 = keep everything
		PruneInterval: 1 * time.Hour, // only applies when RetentionDays > 0
	}, nil
}

// Close releases the database handle.
func (l *Logger) Close() error { return l.db.Close() }

// Record inserts a new audit entry.
func (l *Logger) Record(ctx context.Context, e Entry) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO audit_log (cluster_name, operation, operator, timestamp, result, detail)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		e.ClusterName, e.Operation, e.Operator, e.Timestamp, e.Result, e.Detail)
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	metrics.AuditRecordTotal.WithLabelValues(e.ClusterName, e.Result).Inc()
	return nil
}

// Query returns the most recent entries for a cluster (newest first).
func (l *Logger) Query(ctx context.Context, clusterName string, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 100
	}
	var out []Entry
	err := l.db.SelectContext(ctx, &out,
		`SELECT id, cluster_name, operation, operator, timestamp, result, detail
		 FROM audit_log WHERE cluster_name = $1 ORDER BY timestamp DESC LIMIT $2`,
		clusterName, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}
	return out, nil
}

// Prune deletes entries older than before. Returns the number of rows
// removed.
func (l *Logger) Prune(ctx context.Context, before time.Time) (int64, error) {
	res, err := l.db.ExecContext(ctx,
		`DELETE FROM audit_log WHERE timestamp < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("prune audit: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	metrics.AuditPruneTotal.Add(float64(n))
	return n, nil
}

// RowCount returns the total number of rows in the audit table. Used to
// update the audit_rows gauge.
func (l *Logger) RowCount(ctx context.Context) (int64, error) {
	var n int64
	if err := l.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM audit_log`); err != nil {
		return 0, fmt.Errorf("count audit: %w", err)
	}
	return n, nil
}

// Start runs the periodic pruner until ctx is cancelled. No-op when
// RetentionDays is 0. Satisfies manager.Runnable so it can be registered
// alongside the other background loops.
func (l *Logger) Start(ctx context.Context) error {
	log := logf.FromContext(ctx).WithName("audit-pruner")
	if l.RetentionDays <= 0 {
		log.Info("retention disabled; pruner is a no-op")
		<-ctx.Done()
		return nil
	}
	ticker := time.NewTicker(l.PruneInterval)
	defer ticker.Stop()
	// Run once on startup so operators can see the effect without waiting
	// a full interval.
	l.tick(ctx, log)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			l.tick(ctx, log)
		}
	}
}

func (l *Logger) tick(ctx context.Context, log interface {
	Error(error, string, ...any)
	Info(string, ...any)
}) {
	cutoff := time.Now().UTC().Add(-time.Duration(l.RetentionDays) * 24 * time.Hour)
	n, err := l.Prune(ctx, cutoff)
	if err != nil {
		log.Error(err, "prune failed")
		return
	}
	if n > 0 {
		log.Info("pruned audit rows", "count", n, "cutoff", cutoff.Format(time.RFC3339))
	}
	if count, err := l.RowCount(ctx); err == nil {
		metrics.AuditRows.Set(float64(count))
	}
}
