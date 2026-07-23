// Package tools defines the Tool interface and shared result types for Atlas.
package tools

import (
	"context"
	"time"

	"github.com/Yashh56/atlas/internal/session"
)

// ToolResult holds the outcome of a single tool execution.
type ToolResult struct {
	Success  bool
	Output     string
	Error      string
	Duration   time.Duration
	TokenUsage *TokenUsage
}

// TokenUsage tracks LLM usage for a tool execution.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// Tool is the interface every Atlas tool must implement.
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string
	// Execute runs the tool against the given session and returns the result.
	// Implementations must respect ctx cancellation/deadline.
	Execute(ctx context.Context, s *session.Session) (ToolResult, error)
}
