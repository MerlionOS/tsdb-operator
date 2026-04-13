package audit

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestEntryDefaultsTimestamp(t *testing.T) {
	e := Entry{ClusterName: "c", Operation: "create", Operator: "u", Result: "success"}
	if !e.Timestamp.IsZero() {
		t.Fatalf("expected zero timestamp, got %v", e.Timestamp)
	}
	e.Timestamp = time.Now().UTC()
	if e.Timestamp.IsZero() {
		t.Fatal("timestamp should be set")
	}
}

func newMockLogger(t *testing.T) (*Logger, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return &Logger{db: sqlx.NewDb(db, "postgres")}, mock, func() { _ = db.Close() }
}

func TestPruneDeletesOldRows(t *testing.T) {
	l, mock, cleanup := newMockLogger(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM audit_log WHERE timestamp < $1`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 17))

	n, err := l.Prune(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if n != 17 {
		t.Fatalf("want 17 rows pruned, got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRowCount(t *testing.T) {
	l, mock, cleanup := newMockLogger(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM audit_log`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	n, err := l.RowCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 42 {
		t.Fatalf("want 42, got %d", n)
	}
}

func TestStartNoopWhenRetentionDisabled(t *testing.T) {
	l, _, cleanup := newMockLogger(t)
	defer cleanup()
	l.RetentionDays = 0

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- l.Start(ctx) }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not exit when ctx cancelled")
	}
}
