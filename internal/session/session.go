// Package session manages Atlas agent sessions.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session represents a single Atlas deployment session.
type Session struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"` // see internal/state for valid values
	Goal      string    `json:"goal"`
	Workspace string    `json:"workspace_path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// New creates a new Session with a unique ID and status "created".
func New(goal, workspacePath string) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:        generateID(),
		Status:    "created",
		Goal:      goal,
		Workspace: workspacePath,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Save atomically persists the session to <sessionsDir>/<id>/session.json.
// It writes to a .tmp file first and then renames to prevent corruption on crash.
// UpdatedAt is refreshed before writing.
func (s *Session) Save(sessionsDir string) error {
	dir := filepath.Join(sessionsDir, s.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("session: creating directory %s: %w", dir, err)
	}

	s.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshalling %s: %w", s.ID, err)
	}

	final := filepath.Join(dir, "session.json")
	tmp := final + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("session: writing tmp file %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, final); err != nil {
		// Best-effort cleanup of the tmp file.
		_ = os.Remove(tmp)
		return fmt.Errorf("session: renaming %s → %s: %w", tmp, final, err)
	}

	return nil
}

// Load reads the session identified by id from <sessionsDir>/<id>/session.json.
func Load(sessionsDir, id string) (*Session, error) {
	path := filepath.Join(sessionsDir, id, "session.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("session: reading %s: %w", path, err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("session: parsing %s: %w", path, err)
	}
	return &s, nil
}

// generateID returns a unique session ID of the form "sess_<16 hex chars>".
func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("session: crypto/rand unavailable: %v", err))
	}
	return "sess_" + hex.EncodeToString(b)
}
