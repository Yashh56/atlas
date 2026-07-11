// Package state defines the status constants and shared persistence helpers
// used across Atlas components.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Session status values.
const (
	StatusCreated = "created"
	StatusRunning = "running"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// SaveJSON atomically writes v (marshalled as indented JSON) to
// filepath.Join(dir, filename). The write goes to a .tmp file first and is
// then renamed over the final path, so a crash mid-write never corrupts the
// last-known-good file.
func SaveJSON(dir, filename string, v any) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("state: mkdir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshalling %s: %w", filename, err)
	}

	final := filepath.Join(dir, filename)
	tmp := final + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("state: writing %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("state: renaming %s → %s: %w", tmp, final, err)
	}

	return nil
}

// LoadJSON reads filename from dir and unmarshals JSON into v.
func LoadJSON(dir, filename string, v any) error {
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("state: reading %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("state: parsing %s: %w", path, err)
	}
	return nil
}
