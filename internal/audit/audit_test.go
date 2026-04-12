package audit

import (
	"testing"
	"time"
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
