package observability

import (
	"errors"
	"testing"
	"time"
)

func TestRegistryRecord(t *testing.T) {
	r := NewRegistry()
	r.RecordSuccess("inventory", 12*time.Millisecond)
	r.RecordError("stats", time.Millisecond, errors.New("boom"))
	snap := r.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("%+v", snap)
	}
	if snap[0].Name != "inventory" || !snap[0].OK {
		t.Fatalf("%+v", snap[0])
	}
	if snap[1].Name != "stats" || snap[1].OK || snap[1].LastError != "boom" {
		t.Fatalf("%+v", snap[1])
	}
}
