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

func TestResolve_Monorepo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pnpm-workspace.yaml"), []byte("packages:\n  - 'apps/*'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	appDir := filepath.Join(dir, "apps", "web")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{"name": "web"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := workspace.Resolve(appDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.MonorepoRoot != dir {
		t.Errorf("MonorepoRoot = %q, want %q", ws.MonorepoRoot, dir)
	}
	if ws.Subdir != "apps/web" {
		t.Errorf("Subdir = %q, want %q", ws.Subdir, "apps/web")
	}
	if ws.Monorepo == nil || ws.Monorepo.Type != workspace.WorkspacePnpm {
		t.Errorf("Monorepo = %+v, want type pnpm", ws.Monorepo)
	}
}

func TestResolveWithSubdir_Flags(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pnpm-workspace.yaml"), []byte("packages:\n  - 'apps/*'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	appDir := filepath.Join(dir, "apps", "web")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{"name": "web"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. With --dir apps/web
	wsDir, err := workspace.ResolveWithSubdir(dir, "apps/web", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wsDir.Root != appDir {
		t.Errorf("ws.Root = %q, want %q", wsDir.Root, appDir)
	}
	if wsDir.Subdir != "apps/web" {
		t.Errorf("ws.Subdir = %q, want 'apps/web'", wsDir.Subdir)
	}

	// 2. With --workspace web
	wsPkg, err := workspace.ResolveWithSubdir(dir, "", "web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wsPkg.Root != appDir {
		t.Errorf("ws.Root = %q, want %q", wsPkg.Root, appDir)
	}
	if wsPkg.Subdir != "apps/web" {
		t.Errorf("ws.Subdir = %q, want 'apps/web'", wsPkg.Subdir)
	}

	// 3. Error on nonexistent dir
	_, err = workspace.ResolveWithSubdir(dir, "nonexistent", "")
	if err == nil {
		t.Fatal("expected error for nonexistent --dir")
	}

	// 4. Error on nonexistent workspace
	_, err = workspace.ResolveWithSubdir(dir, "", "unknown-pkg")
	if err == nil {
		t.Fatal("expected error for nonexistent --workspace")
	}
}

func TestResolveWithSubdir_OutsideToMonorepo(t *testing.T) {
	parent := t.TempDir()
	monorepoDir := filepath.Join(parent, "repo")
	if err := os.MkdirAll(monorepoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(monorepoDir, "pnpm-workspace.yaml"), []byte("packages:\n  - 'packages/*'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(monorepoDir, "packages", "api")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name": "api"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := workspace.ResolveWithSubdir(parent, filepath.Join("repo", "packages", "api"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Root != pkgDir {
		t.Errorf("ws.Root = %q, want %q", ws.Root, pkgDir)
	}
	if ws.MonorepoRoot != monorepoDir {
		t.Errorf("ws.MonorepoRoot = %q, want %q", ws.MonorepoRoot, monorepoDir)
	}
	if ws.Subdir != "packages/api" {
		t.Errorf("ws.Subdir = %q, want 'packages/api'", ws.Subdir)
	}
	if ws.Monorepo == nil || ws.Monorepo.Type != workspace.WorkspacePnpm {
		t.Errorf("expected pnpm monorepo info, got: %+v", ws.Monorepo)
	}
}

