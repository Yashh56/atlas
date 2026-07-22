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

// PatchFile is a tool that writes content to a file by replacing an exact
// string match. It prevents path traversal out of the workspace root.
type PatchFile struct {
	WorkspaceRoot string
	SessionDir    string
	Path          string // relative to WorkspaceRoot
	OldStr        string
	NewStr        string
}

// Name returns the canonical tool identifier.
func (p PatchFile) Name() string { return "patch_file" }

// Execute validates the path, reads the existing file, safely matches and
// replaces OldStr with NewStr exactly once, and writes it back.
func (p PatchFile) Execute(ctx context.Context, _ *session.Session) (ToolResult, error) {
	start := time.Now()

	// 1. Path safety check.
	targetPath, rel, err := ResolveWorkspacePath(p.WorkspaceRoot, p.Path)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("patch_file: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 2. Read existing content.
	contentBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("reading %s: %v", p.Path, err),
			Duration: time.Since(start),
		}, nil
	}
	content := string(contentBytes)

	// 3. Line Ending Detection & Normalization
	// The LLM often outputs "\n" (LF) regardless of the file's original line endings.
	// If the file uses "\r\n" (CRLF), we must format OldStr and NewStr to match it.
	isCRLF := strings.Contains(content, "\r\n")

	oldStrNormalized := p.OldStr
	newStrNormalized := p.NewStr

	// First, normalize everything in OldStr/NewStr to standard LF just to be safe from mixed endings
	oldStrNormalized = strings.ReplaceAll(oldStrNormalized, "\r\n", "\n")
	newStrNormalized = strings.ReplaceAll(newStrNormalized, "\r\n", "\n")

	// Then, if the original file is CRLF, convert OldStr/NewStr to CRLF
	if isCRLF {
		oldStrNormalized = strings.ReplaceAll(oldStrNormalized, "\n", "\r\n")
		newStrNormalized = strings.ReplaceAll(newStrNormalized, "\n", "\r\n")
	}

	// 4. Exact-match counting on raw file
	count := strings.Count(content, oldStrNormalized)
	switch {
	case count == 0:
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("old_str not found in file — the model likely paraphrased instead of copying exactly (old_str was: %q)", p.OldStr),
			Duration: time.Since(start),
		}, nil
	case count > 1:
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("old_str found %d times, must be unique — the model's snippet wasn't specific enough (old_str was: %q)", count, p.OldStr),
			Duration: time.Since(start),
		}, nil
	}

	// 5. Apply the patch strictly
	patched := strings.Replace(content, oldStrNormalized, newStrNormalized, 1)

	// 5.5 Snapshot the original file before modifying it for the first time
	if p.SessionDir != "" {
		backupPath := filepath.Join(p.SessionDir, "backups", rel)
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err == nil {
				_ = os.WriteFile(backupPath, contentBytes, 0o644)
			}
		}
	}

	// 6. Write back to disk
	if err := os.WriteFile(targetPath, []byte(patched), 0o644); err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("writing %s: %v", p.Path, err),
			Duration: time.Since(start),
		}, nil
	}

	output := fmt.Sprintf("Patched %s", rel)
	return ToolResult{
		Success:  true,
		Output:   output,
		Duration: time.Since(start),
	}, nil
}
