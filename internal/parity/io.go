package parity

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadFile reads a canonical parity snapshot JSON.
func LoadFile(path string) (Snapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return Snapshot{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if s.SchemaVersion != 0 && s.SchemaVersion != SchemaVersion {
		return Snapshot{}, fmt.Errorf("%s: unsupported schemaVersion %d (want %d)", path, s.SchemaVersion, SchemaVersion)
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	return s, nil
}

// WriteFile writes a canonical parity snapshot JSON (pretty).
func WriteFile(path string, s Snapshot) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
