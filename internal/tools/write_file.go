package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/session"
)

// WriteFile is a tool that writes content to a file. This is the first mutating
// filesystem tool. It prevents path traversal out of the workspace root.
type WriteFile struct {
	WorkspaceRoot string
	Path          string // relative to WorkspaceRoot
	Content       string
}

// Name returns the canonical tool identifier.
func (w WriteFile) Name() string { return "write_file" }

// Execute validates the path and writes the file. It will create parent
// directories if they do not exist and overwrite the file if it does.
func (w WriteFile) Execute(ctx context.Context, _ *session.Session) (ToolResult, error) {
	start := time.Now()

	// 1. Path safety check.
	// We want to ensure Path resolved against WorkspaceRoot does not escape.
	// Clean both paths to normalize them.
	absRoot, err := filepath.Abs(w.WorkspaceRoot)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("write_file: failed to get absolute path for workspace: %v", err),
			Duration: time.Since(start),
		}, nil
	}
	absRoot = filepath.Clean(absRoot)

	// Build the absolute target path.
	targetPath, err := filepath.Abs(filepath.Join(absRoot, w.Path))
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("write_file: failed to resolve target path: %v", err),
			Duration: time.Since(start),
		}, nil
	}
	targetPath = filepath.Clean(targetPath)

	// Check if targetPath escapes absRoot.
	// A simple way is to check if targetPath starts with absRoot + separator
	// (or exactly equals absRoot, though writing to the root dir itself is likely an error).
	rel, err := filepath.Rel(absRoot, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("write_file: path %q attempts to escape workspace root", w.Path),
			Duration: time.Since(start),
		}, nil
	}

	// 2. Create parent directories.
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("write_file: failed to create parent directories: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 3. Write file.
	contentBytes := []byte(w.Content)
	if err := os.WriteFile(targetPath, contentBytes, 0o644); err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("write_file: failed to write file: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 4. Return success audit trail.
	output := fmt.Sprintf("Wrote %d bytes to %s", len(contentBytes), rel)
	return ToolResult{
		Success:  true,
		Output:   output,
		Duration: time.Since(start),
	}, nil
}
