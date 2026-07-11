package orchestrator

import (
	"time"

	"github.com/Yashh56/atlas/internal/state"
)

// BuildState is written by RunBuildCommand. It is the fifth context file.
type BuildState struct {
	Command   string    `json:"command"`
	ExitCode  int       `json:"exit_code"`
	DurationMs int64   `json:"duration_ms"`
	LogPath   string    `json:"log_path"`
	StartedAt time.Time `json:"started_at"`
}

const buildFile = "build.json"

// SaveBuild atomically writes the build state to dir/build.json.
func SaveBuild(dir string, b *BuildState) error {
	return state.SaveJSON(dir, buildFile, b)
}

// LoadBuild reads build.json from dir.
func LoadBuild(dir string) (*BuildState, error) {
	var b BuildState
	if err := state.LoadJSON(dir, buildFile, &b); err != nil {
		return nil, err
	}
	return &b, nil
}
