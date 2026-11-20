package railway

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Yashh56/atlas/internal/deploy"
)

var execCommandContext = exec.CommandContext

// Logs streams logs from railway logs.
func (p *RailwayProvider) Logs(ctx context.Context, d *deploy.Deployment, w io.Writer, follow bool) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "--follow")
	}

	cmdName := "railway"
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath(cmdName); err == nil {
			ext := strings.ToLower(filepath.Ext(p))
			if ext == ".cmd" || ext == ".bat" {
				args = append([]string{"/d", "/c", p}, args...)
				cmdName = "cmd.exe"
			}
		}
	}

	cmd := execCommandContext(ctx, cmdName, args...)
	if d != nil && d.WorkspaceRoot != "" {
		cmd.Dir = d.WorkspaceRoot
	}
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}
