package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveWorkspacePath takes a workspace root and a relative path, and safely
// resolves them into an absolute path, ensuring that the target does not escape
// the workspace root directory (path traversal guard).
// It returns the absolute path, and the cleaned relative path for logging.
func ResolveWorkspacePath(workspaceRoot, targetPath string) (absPath, relPath string, err error) {
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", "", fmt.Errorf("failed to get absolute path for workspace: %w", err)
	}
	absRoot = filepath.Clean(absRoot)

	// Build the absolute target path.
	absTarget, err := filepath.Abs(filepath.Join(absRoot, targetPath))
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve target path: %w", err)
	}
	absTarget = filepath.Clean(absTarget)

	// Check if absTarget escapes absRoot.
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return "", "", fmt.Errorf("path %q attempts to escape workspace root", targetPath)
	}

	return absTarget, rel, nil
}
