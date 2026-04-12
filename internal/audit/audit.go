// Package audit persists cluster-management operations to a PostgreSQL table.
package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
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
	db *sqlx.DB
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
	return &Logger{db: db}, nil
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
