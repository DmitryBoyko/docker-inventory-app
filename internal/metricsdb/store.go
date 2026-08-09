package metricsdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Point is one series sample for charts / API.
type Point struct {
	T     int64   `json:"t"` // unix milliseconds
	CPU   float64 `json:"cpu"`
	Mem   int64   `json:"mem"`
	NetRx int64   `json:"netRx,omitempty"`
	NetTx int64   `json:"netTx,omitempty"`
}

// ContainerSample is one container stats row ready to persist.
type ContainerSample struct {
	ID    string
	Name  string
	Stack string
	CPU   float64
	Mem   int64
	NetRx int64
	NetTx int64
}

// Store is a SQLite-backed historical metrics DB (ADR-015).
type Store struct {
	db             *sql.DB
	path           string
	retention      time.Duration
	sampleInterval time.Duration

	mu        sync.Mutex
	lastWrite map[string]time.Time
}

// Options configures Open.
type Options struct {
	Path           string
	Retention      time.Duration
	SampleInterval time.Duration
}

// ParsePath returns path and whether metrics are enabled.
// Empty, "off", "false", "0", "none", "disabled" ⇒ disabled.
func ParsePath(raw string) (path string, enabled bool) {
	s := strings.TrimSpace(raw)
	switch strings.ToLower(s) {
	case "", "off", "false", "0", "none", "disabled":
		return "", false
	default:
		return s, true
	}
}

// Open creates/opens the SQLite file. Caller must Close.
func Open(opts Options) (*Store, error) {
	path, ok := ParsePath(opts.Path)
	if !ok {
		return nil, fmt.Errorf("metrics db disabled")
	}
	if opts.Retention <= 0 {
		opts.Retention = 24 * time.Hour
	}
	if opts.SampleInterval <= 0 {
		opts.SampleInterval = 10 * time.Second
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, fmt.Errorf("metrics db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	s := &Store{
		db:             db,
		path:           path,
		retention:      opts.Retention,
		sampleInterval: opts.SampleInterval,
		lastWrite:      map[string]time.Time{},
	}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS host_samples (
  host TEXT NOT NULL,
  ts_ms INTEGER NOT NULL,
  cpu_percent REAL NOT NULL,
  memory_bytes INTEGER NOT NULL,
  network_rx_bytes INTEGER NOT NULL DEFAULT 0,
  network_tx_bytes INTEGER NOT NULL DEFAULT 0,
  container_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (host, ts_ms)
);
CREATE TABLE IF NOT EXISTS container_samples (
  host TEXT NOT NULL,
  container_id TEXT NOT NULL,
  container_name TEXT NOT NULL DEFAULT '',
  stack TEXT NOT NULL DEFAULT '',
  ts_ms INTEGER NOT NULL,
  cpu_percent REAL NOT NULL,
  memory_bytes INTEGER NOT NULL,
  network_rx_bytes INTEGER NOT NULL DEFAULT 0,
  network_tx_bytes INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (host, container_id, ts_ms)
);
CREATE INDEX IF NOT EXISTS idx_host_samples_ts ON host_samples(ts_ms);
CREATE INDEX IF NOT EXISTS idx_container_samples_lookup ON container_samples(host, container_id, ts_ms);
CREATE INDEX IF NOT EXISTS idx_container_samples_ts ON container_samples(ts_ms);
`)
	return err
}

// Path returns the configured DB file path.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Retention returns configured retention.
func (s *Store) Retention() time.Duration {
	if s == nil {
		return 0
	}
	return s.retention
}

// SampleInterval returns write cadence.
func (s *Store) SampleInterval() time.Duration {
	if s == nil {
		return 0
	}
	return s.sampleInterval
}

// Close releases the DB.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Record persists a downsampled host rollup + per-container rows when due.
func (s *Store) Record(host string, at time.Time, containers []ContainerSample) error {
	if s == nil || s.db == nil {
		return nil
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "default"
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}

	s.mu.Lock()
	last := s.lastWrite[host]
	if !last.IsZero() && at.Sub(last) < s.sampleInterval {
		s.mu.Unlock()
		return nil
	}
	s.lastWrite[host] = at
	s.mu.Unlock()

	var cpu float64
	var mem, rx, tx int64
	for _, c := range containers {
		cpu += c.CPU
		mem += c.Mem
		rx += c.NetRx
		tx += c.NetTx
	}
	ts := at.UnixMilli()

	txDB, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = txDB.Rollback() }()

	if _, err := txDB.Exec(`
INSERT OR REPLACE INTO host_samples(host, ts_ms, cpu_percent, memory_bytes, network_rx_bytes, network_tx_bytes, container_count)
VALUES(?,?,?,?,?,?,?)`,
		host, ts, cpu, mem, rx, tx, len(containers),
	); err != nil {
		return err
	}
	for _, c := range containers {
		id := c.ID
		if id == "" {
			continue
		}
		if _, err := txDB.Exec(`
INSERT OR REPLACE INTO container_samples(host, container_id, container_name, stack, ts_ms, cpu_percent, memory_bytes, network_rx_bytes, network_tx_bytes)
VALUES(?,?,?,?,?,?,?,?,?)`,
			host, id, c.Name, c.Stack, ts, c.CPU, c.Mem, c.NetRx, c.NetTx,
		); err != nil {
			return err
		}
	}
	return txDB.Commit()
}

// QueryHost returns host rollup points in [from,to], optionally bucketed by step.
func (s *Store) QueryHost(host string, from, to time.Time, step time.Duration) ([]Point, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("metrics disabled")
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "default"
	}
	from, to, stepMs := normalizeRange(from, to, step, s.sampleInterval)
	if stepMs <= int64(s.sampleInterval/time.Millisecond) {
		rows, err := s.db.Query(`
SELECT ts_ms, cpu_percent, memory_bytes, network_rx_bytes, network_tx_bytes
FROM host_samples
WHERE host=? AND ts_ms>=? AND ts_ms<=?
ORDER BY ts_ms ASC`, host, from.UnixMilli(), to.UnixMilli())
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanPoints(rows)
	}
	rows, err := s.db.Query(`
SELECT (ts_ms / ?) * ? AS bucket,
       AVG(cpu_percent), AVG(memory_bytes),
       AVG(network_rx_bytes), AVG(network_tx_bytes)
FROM host_samples
WHERE host=? AND ts_ms>=? AND ts_ms<=?
GROUP BY bucket
ORDER BY bucket ASC`, stepMs, stepMs, host, from.UnixMilli(), to.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAggPoints(rows)
}

// QueryContainer returns per-container points. id may be full or short prefix.
func (s *Store) QueryContainer(host, id string, from, to time.Time, step time.Duration) ([]Point, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("metrics disabled")
	}
	host = strings.TrimSpace(host)
	id = strings.TrimSpace(id)
	if host == "" {
		host = "default"
	}
	if id == "" {
		return nil, fmt.Errorf("container id required")
	}
	from, to, stepMs := normalizeRange(from, to, step, s.sampleInterval)

	resolved, err := s.resolveContainerID(host, id)
	if err != nil {
		return nil, err
	}

	if stepMs <= int64(s.sampleInterval/time.Millisecond) {
		rows, err := s.db.Query(`
SELECT ts_ms, cpu_percent, memory_bytes, network_rx_bytes, network_tx_bytes
FROM container_samples
WHERE host=? AND container_id=? AND ts_ms>=? AND ts_ms<=?
ORDER BY ts_ms ASC`, host, resolved, from.UnixMilli(), to.UnixMilli())
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanPoints(rows)
	}
	rows, err := s.db.Query(`
SELECT (ts_ms / ?) * ? AS bucket,
       AVG(cpu_percent), AVG(memory_bytes),
       AVG(network_rx_bytes), AVG(network_tx_bytes)
FROM container_samples
WHERE host=? AND container_id=? AND ts_ms>=? AND ts_ms<=?
GROUP BY bucket
ORDER BY bucket ASC`, stepMs, stepMs, host, resolved, from.UnixMilli(), to.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAggPoints(rows)
}

func (s *Store) resolveContainerID(host, id string) (string, error) {
	var full string
	err := s.db.QueryRow(`
SELECT container_id FROM container_samples
WHERE host=? AND (container_id=? OR container_id LIKE ?)
ORDER BY ts_ms DESC LIMIT 1`, host, id, id+"%").Scan(&full)
	if err == sql.ErrNoRows {
		return id, nil
	}
	return full, err
}

// Prune deletes samples older than retention.
func (s *Store) Prune(now time.Time) (int64, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.UTC().Add(-s.retention).UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	r1, err := tx.Exec(`DELETE FROM host_samples WHERE ts_ms < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	r2, err := tx.Exec(`DELETE FROM container_samples WHERE ts_ms < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	n1, _ := r1.RowsAffected()
	n2, _ := r2.RowsAffected()
	return n1 + n2, nil
}

// RunPruner periodically deletes expired rows until ctx is done.
func (s *Store) RunPruner(ctx context.Context) {
	if s == nil {
		return
	}
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	_, _ = s.Prune(time.Now().UTC())
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_, _ = s.Prune(time.Now().UTC())
		}
	}
}

func normalizeRange(from, to time.Time, step, sampleInterval time.Duration) (time.Time, time.Time, int64) {
	now := time.Now().UTC()
	if to.IsZero() {
		to = now
	} else {
		to = to.UTC()
	}
	if from.IsZero() {
		from = to.Add(-time.Hour)
	} else {
		from = from.UTC()
	}
	if !from.Before(to) {
		from = to.Add(-time.Hour)
	}
	if step <= 0 {
		span := to.Sub(from)
		switch {
		case span <= time.Hour:
			step = sampleInterval
		case span <= 6*time.Hour:
			step = time.Minute
		default:
			step = 5 * time.Minute
		}
	}
	stepMs := step.Milliseconds()
	if stepMs < 1 {
		stepMs = 1
	}
	return from, to, stepMs
}

func scanPoints(rows *sql.Rows) ([]Point, error) {
	out := make([]Point, 0, 128)
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.T, &p.CPU, &p.Mem, &p.NetRx, &p.NetTx); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanAggPoints(rows *sql.Rows) ([]Point, error) {
	out := make([]Point, 0, 128)
	for rows.Next() {
		var (
			bucket int64
			cpu    float64
			mem    float64
			rx     float64
			tx     float64
		)
		if err := rows.Scan(&bucket, &cpu, &mem, &rx, &tx); err != nil {
			return nil, err
		}
		out = append(out, Point{
			T:     bucket,
			CPU:   cpu,
			Mem:   int64(mem + 0.5),
			NetRx: int64(rx + 0.5),
			NetTx: int64(tx + 0.5),
		})
	}
	return out, rows.Err()
}
