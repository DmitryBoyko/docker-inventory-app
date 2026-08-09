package snapshots

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store persists snapshots as JSON files under Dir.
type Store struct {
	Dir string
	mu  sync.Mutex
}

// NewStore creates a filesystem-backed snapshot store.
func NewStore(dir string) (*Store, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("snapshots dir required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}

// Save writes a snapshot to disk.
func (s *Store) Save(snap Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if snap.ID == "" {
		return fmt.Errorf("snapshot id required")
	}
	if err := validateID(snap.ID); err != nil {
		return err
	}
	path := s.path(snap.ID)
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Get loads a snapshot by id.
func (s *Store) Get(id string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateID(id); err != nil {
		return Snapshot{}, err
	}
	raw, err := os.ReadFile(s.path(id))
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

// List returns metas newest-first.
func (s *Store) List() ([]Meta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}
	out := make([]Meta, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.Dir, e.Name()))
		if err != nil {
			continue
		}
		var snap Snapshot
		if err := json.Unmarshal(raw, &snap); err != nil {
			continue
		}
		out = append(out, snap.Meta)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

// Delete removes a snapshot.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateID(id); err != nil {
		return err
	}
	return os.Remove(s.path(id))
}

func (s *Store) path(id string) string {
	return filepath.Join(s.Dir, id+".json")
}

func validateID(id string) error {
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\:`) {
		return fmt.Errorf("invalid snapshot id")
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == 'T' || r == 'Z' {
			continue
		}
		return fmt.Errorf("invalid snapshot id char")
	}
	return nil
}

// NewID returns a filesystem-safe UTC timestamp id.
func NewID(now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.UTC().Format("20060102T150405.000Z")
}
