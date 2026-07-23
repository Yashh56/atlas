// Package orchestrator owns the planner, project, and deployment context files
// that live alongside session.json in <workspace>/.atlas/sessions/<id>/.
package orchestrator

import (
	"time"

	"github.com/Yashh56/atlas/internal/state"
	"github.com/Yashh56/atlas/internal/tools"
)

// RetryInfo tracks retry state for a single pipeline step.
type RetryInfo struct {
	Count int `json:"count"`
	Max   int `json:"max"`
}

// ErrorInfo captures the last error that occurred during a pipeline run.
type ErrorInfo struct {
	Step       *string    `json:"step"`
	Message    *string    `json:"message"`
	OccurredAt *time.Time `json:"occurred_at"`
}

// PlannerState tracks the goal and pipeline progress. Owned by orchestrator.
type PlannerState struct {
	Goal        string               `json:"goal"`
	CurrentStep string               `json:"current_step"`
	Completed   []string             `json:"completed"`
	Pending     []string             `json:"pending"`
	Failed      []string             `json:"failed"`
	Retries     map[string]RetryInfo `json:"retries"`
	Error       ErrorInfo            `json:"error"`
	TokenUsage  tools.TokenUsage     `json:"token_usage"`
}

// AddUsage accumulates the token usage from a tool execution.
func (p *PlannerState) AddUsage(u *tools.TokenUsage) {
	if u == nil {
		return
	}
	p.TokenUsage.InputTokens += u.InputTokens
	p.TokenUsage.OutputTokens += u.OutputTokens
	p.TokenUsage.TotalTokens += u.TotalTokens
}

const plannerFile = "planner.json"

// NewPlanner returns a PlannerState initialised with the given goal and
// default retry limits.
func NewPlanner(goal string) *PlannerState {
	return &PlannerState{
		Goal:      goal,
		Completed: []string{},
		Pending:   []string{"analyze_project", "determine_build_command", "run_build"},
		Failed:    []string{},
		Retries: map[string]RetryInfo{
			"fix_and_rebuild": {Count: 0, Max: 4},
		},
		Error: ErrorInfo{},
	}
}

// SavePlanner atomically writes the planner state to dir/planner.json.
func SavePlanner(dir string, p *PlannerState) error {
	return state.SaveJSON(dir, plannerFile, p)
}

// LoadPlanner reads planner.json from dir.
func LoadPlanner(dir string) (*PlannerState, error) {
	var p PlannerState
	if err := state.LoadJSON(dir, plannerFile, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
