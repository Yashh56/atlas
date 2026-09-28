package netlify

import (
	"context"
	"io"
	"os/exec"

	"github.com/Yashh56/atlas/internal/deploy"
)

func (p *NetlifyProvider) Logs(ctx context.Context, d *deploy.Deployment, w io.Writer, follow bool) error {
	args := []string{"logs"}

	if follow {
		args = append(args, "--follow")
	}

	cmd := exec.CommandContext(ctx, "netlify", args...)
	cmd.Dir = d.WorkspaceRoot
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}
