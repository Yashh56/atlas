// Package workspace resolves and describes a project working directory.
package workspace

import (
	"errors"
	"os"
	"path/filepath"
)

// Workspace describes a project directory understood by Atlas.
type Workspace struct {
	Root    string // absolute path to the project directory
	GitRoot string // absolute path to nearest .git ancestor; "" if none
	Exists  bool
}

// Resolve returns a Workspace for the given inputPath.
// If the path does not exist on disk, Exists is false and no error is returned.
// GitRoot is determined by walking up the directory tree looking for a .git entry;
// no external processes are invoked.
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

	return &Workspace{
		Root:    root,
		GitRoot: gitRoot,
		Exists:  true,
	}, nil
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
