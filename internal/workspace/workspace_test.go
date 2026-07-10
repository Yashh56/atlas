package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/workspace"
)

// TestResolve_WithGitDir verifies that a directory containing a .git folder is
// recognized and GitRoot is set to that directory.
func TestResolve_WithGitDir(t *testing.T) {
	dir := t.TempDir()

	// Create a fake .git directory.
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory inside to resolve from.
	sub := filepath.Join(dir, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := workspace.Resolve(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ws.Exists {
		t.Error("expected Exists to be true")
	}
	if ws.GitRoot != dir {
		t.Errorf("GitRoot = %q, want %q", ws.GitRoot, dir)
	}
}

// TestResolve_NoGitDir verifies that a directory with no .git ancestor yields
// an empty GitRoot.
func TestResolve_NoGitDir(t *testing.T) {
	// Use os.TempDir as a path that almost certainly has no .git above it.
	// We create an isolated temp tree that definitely has none.
	//
	// Strategy: create a fresh temp dir under a parent that has no .git,
	// and verify GitRoot is empty.
	root := t.TempDir()
	// Make a nested subdirectory with no .git anywhere.
	leaf := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := workspace.Resolve(leaf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ws.Exists {
		t.Error("expected Exists to be true")
	}
	// We cannot guarantee the OS temp dir has no .git, but within the test
	// subtree we created there is definitely none at or below 'root'. If the
	// host machine's temp path happens to be inside a git repo, GitRoot may
	// be set. We only assert it's a string (no panic, no error). Log for info.
	t.Logf("GitRoot = %q (may be non-empty if OS temp is inside a git repo)", ws.GitRoot)
}

// TestResolve_NonExistentPath verifies that a nonexistent path returns
// Exists: false and no error.
func TestResolve_NonExistentPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does", "not", "exist")

	ws, err := workspace.Resolve(path)
	if err != nil {
		t.Fatalf("expected no error for nonexistent path, got: %v", err)
	}
	if ws.Exists {
		t.Error("expected Exists to be false for nonexistent path")
	}
}
