// Package workspace resolves and describes a project working directory.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Workspace describes a project directory understood by Atlas.
type Workspace struct {
	Root         string        // target project directory (where build/app lives)
	GitRoot      string        // nearest .git directory; "" if none
	MonorepoRoot string        // root of monorepo if inside one; "" if none
	Monorepo     *MonorepoInfo // monorepo info, if applicable
	Subdir       string        // relative path from MonorepoRoot/GitRoot to Root
	Exists       bool
}

// Resolve returns a Workspace for the given inputPath.
// If the path does not exist on disk, Exists is false and no error is returned.
// GitRoot is determined by walking up the directory tree looking for a .git entry;
// no external processes are invoked.
// If inside a monorepo, MonorepoRoot, Monorepo, and Subdir are populated.
func Resolve(inputPath string) (*Workspace, error) {
	abs, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Workspace{Root: abs, Exists: false}, nil
		}
		return nil, err
	}

	// Resolve root to an absolute directory path.
	root := abs
	if !info.IsDir() {
		root = filepath.Dir(abs)
	}

	gitRoot := findGitRoot(root)

	ws := &Workspace{
		Root:    root,
		GitRoot: gitRoot,
		Exists:  true,
	}

	mr := DetectMonorepo(root, gitRoot)
	if mr != nil {
		ws.MonorepoRoot = mr.Root
		ws.Monorepo = mr
		if rel, err := filepath.Rel(mr.Root, root); err == nil && rel != "." {
			ws.Subdir = filepath.ToSlash(rel)
		}
	} else if gitRoot != "" {
		if rel, err := filepath.Rel(gitRoot, root); err == nil && rel != "." {
			ws.Subdir = filepath.ToSlash(rel)
		}
	}

	return ws, nil
}

// ResolveWithSubdir resolves the workspace given an inputPath, an optional subdirectory
// path (--dir), and an optional monorepo package name (--workspace).
func ResolveWithSubdir(inputPath, dirFlag, workspaceFlag string) (*Workspace, error) {
	ws, err := Resolve(inputPath)
	if err != nil {
		return nil, err
	}
	if !ws.Exists {
		return ws, nil
	}

	if workspaceFlag != "" {
		if ws.Monorepo == nil {
			return nil, fmt.Errorf("--workspace flag %q specified, but no monorepo detected at %s", workspaceFlag, ws.Root)
		}
		pkg, err := ResolveWorkspacePackage(ws.Monorepo.Root, workspaceFlag)
		if err != nil {
			return nil, err
		}
		ws.Root = pkg.AbsPath
	} else if dirFlag != "" {
		targetDir := dirFlag
		if !filepath.IsAbs(targetDir) {
			targetDir = filepath.Join(ws.Root, dirFlag)
		}
		targetDir = filepath.Clean(targetDir)

		info, err := os.Stat(targetDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("target subdirectory %q does not exist", dirFlag)
			}
			return nil, fmt.Errorf("error checking target subdirectory %q: %w", dirFlag, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("target subdirectory %q is not a directory", dirFlag)
		}
		ws.Root = targetDir
	}

	// Recompute Subdir
	if ws.MonorepoRoot != "" {
		rel, err := filepath.Rel(ws.MonorepoRoot, ws.Root)
		if err == nil && rel != "." {
			ws.Subdir = filepath.ToSlash(rel)
		} else {
			ws.Subdir = ""
		}
	} else if ws.GitRoot != "" {
		rel, err := filepath.Rel(ws.GitRoot, ws.Root)
		if err == nil && rel != "." {
			ws.Subdir = filepath.ToSlash(rel)
		} else {
			ws.Subdir = ""
		}
	} else {
		ws.Subdir = ""
	}

	return ws, nil
}

// findGitRoot walks up from dir until it finds a directory containing a .git
// entry. Returns "" if no such directory is found.
func findGitRoot(dir string) string {
	current := dir
	for {
		candidate := filepath.Join(current, ".git")
		if _, err := os.Stat(candidate); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root.
			return ""
		}
		current = parent
	}
}
