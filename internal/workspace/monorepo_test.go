package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/workspace"
)

func writeWorkspaceFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectMonorepo_Pnpm(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(dir, "pnpm-workspace.yaml"), "packages:\n  - 'apps/*'\n  - 'packages/*'\n")
	writeWorkspaceFile(t, filepath.Join(dir, "apps", "web", "package.json"), `{"name": "web"}`)
	writeWorkspaceFile(t, filepath.Join(dir, "apps", "api", "package.json"), `{"name": "@org/api"}`)
	writeWorkspaceFile(t, filepath.Join(dir, "packages", "ui", "package.json"), `{"name": "@org/ui"}`)

	mr := workspace.DetectMonorepo(dir, "")
	if mr == nil {
		t.Fatal("expected monorepo to be detected")
	}
	if mr.Type != workspace.WorkspacePnpm {
		t.Errorf("Type = %q, want %q", mr.Type, workspace.WorkspacePnpm)
	}
	if mr.Root != dir {
		t.Errorf("Root = %q, want %q", mr.Root, dir)
	}
	if len(mr.Packages) != 3 {
		t.Fatalf("len(Packages) = %d, want 3", len(mr.Packages))
	}

	// Test ResolveWorkspacePackage
	pkg, err := workspace.ResolveWorkspacePackage(dir, "web")
	if err != nil || pkg.RelPath != "apps/web" {
		t.Fatalf("resolve web failed: %v, pkg: %+v", err, pkg)
	}

	pkg, err = workspace.ResolveWorkspacePackage(dir, "@org/api")
	if err != nil || pkg.RelPath != "apps/api" {
		t.Fatalf("resolve @org/api failed: %v, pkg: %+v", err, pkg)
	}

	pkg, err = workspace.ResolveWorkspacePackage(dir, "packages/ui")
	if err != nil || pkg.Name != "@org/ui" {
		t.Fatalf("resolve packages/ui failed: %v, pkg: %+v", err, pkg)
	}

	_, err = workspace.ResolveWorkspacePackage(dir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent package")
	}
}

func TestDetectMonorepo_Turbo(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(dir, "turbo.json"), `{}`)
	writeWorkspaceFile(t, filepath.Join(dir, "package.json"), `{"name": "root", "workspaces": ["apps/*"]}`)
	writeWorkspaceFile(t, filepath.Join(dir, "apps", "web", "package.json"), `{"name": "web"}`)

	mr := workspace.DetectMonorepo(dir, "")
	if mr == nil {
		t.Fatal("expected monorepo to be detected")
	}
	if mr.Type != workspace.WorkspaceTurbo {
		t.Errorf("Type = %q, want %q", mr.Type, workspace.WorkspaceTurbo)
	}
}

func TestDetectMonorepo_Nx(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(dir, "nx.json"), `{}`)
	writeWorkspaceFile(t, filepath.Join(dir, "apps", "web", "package.json"), `{"name": "web"}`)

	mr := workspace.DetectMonorepo(dir, "")
	if mr == nil {
		t.Fatal("expected monorepo to be detected")
	}
	if mr.Type != workspace.WorkspaceNx {
		t.Errorf("Type = %q, want %q", mr.Type, workspace.WorkspaceNx)
	}
}

func TestDetectMonorepo_Lerna(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(dir, "lerna.json"), `{}`)
	writeWorkspaceFile(t, filepath.Join(dir, "packages", "pkg-a", "package.json"), `{"name": "pkg-a"}`)

	mr := workspace.DetectMonorepo(dir, "")
	if mr == nil {
		t.Fatal("expected monorepo to be detected")
	}
	if mr.Type != workspace.WorkspaceLerna {
		t.Errorf("Type = %q, want %q", mr.Type, workspace.WorkspaceLerna)
	}
}

func TestDetectMonorepo_WorkspacesLockfiles(t *testing.T) {
	// Yarn
	dirYarn := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(dirYarn, "package.json"), `{"workspaces": ["packages/*"]}`)
	writeWorkspaceFile(t, filepath.Join(dirYarn, "yarn.lock"), "")
	mrYarn := workspace.DetectMonorepo(dirYarn, "")
	if mrYarn == nil || mrYarn.Type != workspace.WorkspaceYarn {
		t.Errorf("expected Yarn workspace, got %+v", mrYarn)
	}

	// Bun
	dirBun := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(dirBun, "package.json"), `{"workspaces": ["packages/*"]}`)
	writeWorkspaceFile(t, filepath.Join(dirBun, "bun.lockb"), "")
	mrBun := workspace.DetectMonorepo(dirBun, "")
	if mrBun == nil || mrBun.Type != workspace.WorkspaceBun {
		t.Errorf("expected Bun workspace, got %+v", mrBun)
	}

	// Npm
	dirNpm := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(dirNpm, "package.json"), `{"workspaces": ["packages/*"]}`)
	writeWorkspaceFile(t, filepath.Join(dirNpm, "package-lock.json"), "")
	mrNpm := workspace.DetectMonorepo(dirNpm, "")
	if mrNpm == nil || mrNpm.Type != workspace.WorkspaceNpm {
		t.Errorf("expected Npm workspace, got %+v", mrNpm)
	}
}

func TestDetectMonorepo_FromSubdirectory(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(root, "pnpm-workspace.yaml"), "packages:\n  - 'apps/*'\n")
	sub := filepath.Join(root, "apps", "web")
	writeWorkspaceFile(t, filepath.Join(sub, "package.json"), `{"name": "web"}`)

	mr := workspace.DetectMonorepo(sub, root)
	if mr == nil {
		t.Fatal("expected monorepo detected walking up to root")
	}
	if mr.Root != root {
		t.Errorf("mr.Root = %q, want %q", mr.Root, root)
	}
}

func TestDiscoverPackages_ExclusionPattern(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, filepath.Join(dir, "pnpm-workspace.yaml"), "packages:\n  - 'packages/*'\n  - '!packages/private'\n")
	writeWorkspaceFile(t, filepath.Join(dir, "packages", "public", "package.json"), `{"name": "public-pkg"}`)
	writeWorkspaceFile(t, filepath.Join(dir, "packages", "private", "package.json"), `{"name": "private-pkg"}`)

	pkgs := workspace.DiscoverPackages(dir)
	var names []string
	for _, p := range pkgs {
		names = append(names, p.Name)
	}

	for _, name := range names {
		if name == "private-pkg" {
			t.Errorf("expected private-pkg to be excluded, but got discovered packages: %v", names)
		}
	}

	foundPublic := false
	for _, name := range names {
		if name == "public-pkg" {
			foundPublic = true
			break
		}
	}
	if !foundPublic {
		t.Errorf("expected public-pkg to be discovered, got: %v", names)
	}
}

