package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Yashh56/atlas/internal/session"
)

const maxReadBytes = 1 << 20 // 1 MB

// ReadFile reads the contents of Path and returns them in ToolResult.Output.
// Files larger than 1 MB are rejected with an error in ToolResult rather than
// a Go error — callers always have a single result path.
type ReadFile struct {
	Path string
}

// Name returns the canonical tool identifier.
func (r ReadFile) Name() string { return "read_file" }

// Execute reads the file at Path and returns its contents.
func (r ReadFile) Execute(_ context.Context, _ *session.Session) (ToolResult, error) {
	start := time.Now()

	info, err := os.Stat(r.Path)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("read_file: stat %s: %v", r.Path, err),
			Duration: time.Since(start),
		}, nil
	}

	if info.Size() > maxReadBytes {
		return ToolResult{
			Success: false,
			Error: fmt.Sprintf(
				"read_file: %s is %d bytes, exceeds 1 MB limit",
				r.Path, info.Size(),
			),
			Duration: time.Since(start),
		}, nil
	}

	data, err := os.ReadFile(r.Path)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("read_file: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	return ToolResult{
		Success:  true,
		Output:   string(data),
		Duration: time.Since(start),
	}, nil
}
