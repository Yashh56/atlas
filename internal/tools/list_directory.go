package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/session"
)

// ListDirectory lists the non-hidden entries of Path (non-recursive).
// Directory entries are suffixed with "/". Hidden entries (names starting
// with ".") are skipped.
type ListDirectory struct {
	Path string
}

// Name returns the canonical tool identifier.
func (l ListDirectory) Name() string { return "list_directory" }

// Execute lists the directory entries and returns them newline-joined in Output.
func (l ListDirectory) Execute(_ context.Context, _ *session.Session) (ToolResult, error) {
	start := time.Now()

	entries, err := os.ReadDir(l.Path)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("list_directory: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue // skip hidden
		}
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}

	return ToolResult{
		Success:  true,
		Output:   strings.Join(names, "\n"),
		Duration: time.Since(start),
	}, nil
}
