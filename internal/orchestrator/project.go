package orchestrator

import (
	"github.com/Yashh56/atlas/internal/state"
)

// GitInfo holds git metadata for the project (populated by a Week 3 tool).
type GitInfo struct {
	Branch    *string `json:"branch"`
	CommitSHA *string `json:"commit_sha"`
	IsClean   *bool   `json:"is_clean"`
	Remote    *string `json:"remote"`
}

// ProjectState is written by AnalyzeProject. Owned by internal/tools via
// orchestrator's SaveProject/LoadProject helpers.
type ProjectState struct {
	Framework      *string `json:"framework"`
	Language       *string `json:"language"`
	Runtime        *string `json:"runtime"`
	PackageManager *string `json:"package_manager"`
	Docker         bool    `json:"docker"`
	Git             GitInfo `json:"git"`
	RenderServiceID *string `json:"render_service_id,omitempty"`
}

const projectFile = "project.json"

// SaveProject atomically writes the project state to dir/project.json.
func SaveProject(dir string, p *ProjectState) error {
	return state.SaveJSON(dir, projectFile, p)
}

// LoadProject reads project.json from dir.
func LoadProject(dir string) (*ProjectState, error) {
	var p ProjectState
	if err := state.LoadJSON(dir, projectFile, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
