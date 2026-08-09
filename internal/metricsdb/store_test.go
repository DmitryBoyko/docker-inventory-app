package metricsdb

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParsePath(t *testing.T) {
	if _, ok := ParsePath("off"); ok {
		t.Fatal("off should disable")
	}
	p, ok := ParsePath("data/metrics.db")
	if !ok || p != "data/metrics.db" {
		t.Fatalf("%q %v", p, ok)
	}
}

func TestRecordAndQuery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.db")
	s, err := Open(Options{
		Path:           path,
		Retention:      time.Hour,
		SampleInterval: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	base := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		at := base.Add(time.Duration(i) * time.Second)
		err := s.Record("default", at, []ContainerSample{
			{ID: "abc123full", Name: "web", Stack: "prod", CPU: float64(i), Mem: int64(1000 * (i + 1)), NetRx: 10, NetTx: 20},
			{ID: "def456full", Name: "db", Stack: "prod", CPU: 0.5, Mem: 500, NetRx: 1, NetTx: 2},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	hostPts, err := s.QueryHost("default", base.Add(-time.Second), base.Add(10*time.Second), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(hostPts) != 5 {
		t.Fatalf("host pts=%d", len(hostPts))
	}
	if hostPts[2].CPU < 2.4 || hostPts[2].CPU > 2.6 {
		t.Fatalf("cpu=%v want ~2.5", hostPts[2].CPU)
	}

	ctrPts, err := s.QueryContainer("default", "abc123", base, base.Add(10*time.Second), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctrPts) != 5 {
		t.Fatalf("ctr pts=%d", len(ctrPts))
	}

	// Downsample skip: same second interval already written — immediate re-record ignored.
	if err := s.Record("default", base.Add(4*time.Second+100*time.Millisecond), []ContainerSample{
		{ID: "abc123full", Name: "web", Stack: "prod", CPU: 99, Mem: 1},
	}); err != nil {
		t.Fatal(err)
	}
	hostPts2, _ := s.QueryHost("default", base, base.Add(10*time.Second), time.Second)
	if len(hostPts2) != 5 {
		t.Fatalf("after skip pts=%d", len(hostPts2))
	}
}

func TestPrune(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(Options{
		Path:           filepath.Join(dir, "p.db"),
		Retention:      time.Hour,
		SampleInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	old := time.Now().UTC().Add(-2 * time.Hour)
	_ = s.Record("default", old, []ContainerSample{{ID: "x", CPU: 1, Mem: 1}})
	// Force write of a fresh sample (reset lastWrite by using new host or wait)
	s.mu.Lock()
	delete(s.lastWrite, "default")
	s.mu.Unlock()
	_ = s.Record("default", time.Now().UTC(), []ContainerSample{{ID: "x", CPU: 2, Mem: 2}})

	n, err := s.Prune(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("pruned=%d", n)
	}
	pts, err := s.QueryHost("default", time.Now().Add(-30*time.Minute), time.Now().Add(time.Minute), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 1 {
		t.Fatalf("remaining=%d", len(pts))
	}
}
