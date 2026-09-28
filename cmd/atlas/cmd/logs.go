package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/deploy/netlify"
	"github.com/Yashh56/atlas/internal/deploy/render"
	"github.com/Yashh56/atlas/internal/deploy/vercel"
	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	logsBuildFlag    bool
	logsFollowFlag   bool
	logsProviderFlag string
	logsSessionID    string
)

var logsCmd = &cobra.Command{
	Use:   "logs [path]",
	Short: "View deployment logs",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().BoolVar(&logsBuildFlag, "build", false, "Show local build logs instead of remote runtime logs")
	logsCmd.Flags().BoolVar(&logsFollowFlag, "follow", false, "Stream logs continuously")
	logsCmd.Flags().StringVar(&logsProviderFlag, "provider", "", "Deployment provider (if overriding session state)")
	logsCmd.Flags().StringVar(&logsSessionID, "session", "", "Specific session UUID to check (default: latest)")
}

func dashboardURL(d *deploy.Deployment) string {
	switch d.Provider {
	case "vercel":
		return "https://vercel.com/dashboard"
	case "render":
		return "https://dashboard.render.com/"
	case "netlify":
		return "https://app.netlify.com/"
	}
	return d.URL
}

func runLogs(_ *cobra.Command, args []string) error {
	var ws *workspace.Workspace
	var err error

	if len(args) == 1 {
		ws, err = workspace.Resolve(args[0])
	} else {
		ws, err = workspace.Resolve(".")
	}
	if err != nil {
		return err
	}

	sessionsDir := filepath.Join(ws.Root, ".atlas", "sessions")
	id := logsSessionID
	if id == "" {
		latestID, err := session.GetLatestID(sessionsDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("no Atlas sessions found. The directory does not have an .atlas folder")
			}
			return fmt.Errorf("could not find latest session: %w", err)
		}
		id = latestID
	}

	sessionDir := session.SessionDir(sessionsDir, id)

	if logsBuildFlag {
		buildState, err := orchestrator.LoadBuild(sessionDir)
		if err != nil || buildState == nil || buildState.LogPath == "" {
			return fmt.Errorf("no local build log available for session %s", id)
		}
		logData, err := os.ReadFile(buildState.LogPath)
		if err != nil {
			return fmt.Errorf("failed to read build log: %w", err)
		}
		fmt.Print(string(logData))
		return nil
	}

	// Runtime logs
	deployState, err := orchestrator.LoadDeployment(sessionDir)
	if err != nil || deployState == nil {
		return fmt.Errorf("no deployment state available for session %s (did the deployment complete?)", id)
	}

	providerName := deployState.Provider
	if logsProviderFlag != "" {
		providerName = logsProviderFlag
	}

	if providerName == "" {
		return fmt.Errorf("could not determine provider")
	}

	var provider deploy.Provider
	switch providerName {
	case "vercel":
		provider = &vercel.VercelProvider{}
	case "render":
		provider = &render.RenderProvider{}
	case "netlify":
		provider = &netlify.NetlifyProvider{}
	default:
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	d := &deploy.Deployment{
		URL:           "",
		Provider:      providerName,
		ProviderRef:   "",
		WorkspaceRoot: ws.Root,
	}
	if deployState.DeploymentURL != nil {
		d.URL = *deployState.DeploymentURL
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cancel()
	}()

	if streamer, ok := provider.(deploy.LogStreamer); ok {
		err := streamer.Logs(ctx, d, os.Stdout, logsFollowFlag)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
	} else {
		fmt.Printf("No log streaming for %s yet — view logs at: %s\n", provider.Name(), dashboardURL(d))
	}

	return nil
}
