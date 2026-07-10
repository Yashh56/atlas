package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
)

func TestSession_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	s := session.New("deploy my app", "/tmp/myapp")
	s.Status = "running"

	if err := s.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := session.Load(dir, s.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ID != s.ID {
		t.Errorf("ID mismatch: got %q, want %q", loaded.ID, s.ID)
	}
	if loaded.Status != s.Status {
		t.Errorf("Status mismatch: got %q, want %q", loaded.Status, s.Status)
	}
	if loaded.Goal != s.Goal {
		t.Errorf("Goal mismatch: got %q, want %q", loaded.Goal, s.Goal)
	}
	if loaded.Workspace != s.Workspace {
		t.Errorf("Workspace mismatch: got %q, want %q", loaded.Workspace, s.Workspace)
	}
	if !loaded.CreatedAt.Equal(s.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", loaded.CreatedAt, s.CreatedAt)
	}
}

func TestSession_DoubleSave_NoStrayTmpFile(t *testing.T) {
	dir := t.TempDir()

	s := session.New("test goal", "/tmp/project")

	if err := s.Save(dir); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	s.Status = "done"

	if err := s.Save(dir); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}

	// Assert no .tmp file remains.
	tmpPath := filepath.Join(dir, s.ID, "session.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("stray .tmp file found at %s after double Save", tmpPath)
	}

	// Also verify the final state is the last save.
	loaded, err := session.Load(dir, s.ID)
	if err != nil {
		t.Fatalf("Load after double Save failed: %v", err)
	}
	if loaded.Status != "done" {
		t.Errorf("Status after double Save = %q, want %q", loaded.Status, "done")
	}
}
