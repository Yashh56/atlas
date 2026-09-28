package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/workspace"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	statusSessionID string
	statusJSON      bool
)

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show the status of the latest or specified Atlas deployment session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&statusSessionID, "session", "", "Specific session UUID to check (default: latest)")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
}

func statusColor(status string) lipgloss.Style {
	switch status {
	case "done":
		return lipgloss.NewStyle().Foreground(cliutil.ColorSuccess)
	case "failed":
		return lipgloss.NewStyle().Foreground(cliutil.ColorError)
	case "running":
		return lipgloss.NewStyle().Foreground(cliutil.ColorHighlight)
	default:
		return lipgloss.NewStyle().Foreground(cliutil.ColorDim)
	}
}

func stepLabel(step string, planner *orchestrator.PlannerState, build *orchestrator.BuildState) string {
	base := strings.ReplaceAll(strings.Title(strings.ReplaceAll(step, "_", " ")), " ", "")

	if step == "run_build" && build != nil {
		if build.ExitCode == 0 {
			return fmt.Sprintf("%s (Success, %dms)", base, build.DurationMs)
		}
		return fmt.Sprintf("%s (Failed, Exit %d)", base, build.ExitCode)
	}

	if step == "fix_and_rebuild" && planner != nil {
		if retries, ok := planner.Retries["fix_and_rebuild"]; ok {
			return fmt.Sprintf("%s (%d/%d fixes applied)", base, retries.Count, retries.Max)
		}
	}

	return base
}

func healthStatus(d *orchestrator.DeploymentState) string {
	if d.HealthCheck.Attempts == 0 {
		return "Pending"
	}
	if d.LastHealthyDeployment != nil {
		return "Healthy"
	}
	if d.HealthCheck.Attempts >= d.HealthCheck.MaxAttempts {
		return "Failed"
	}
	return "Checking..."
}

func timeSince(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	return fmt.Sprintf("%d mins ago", int(d.Minutes()))
}

func runStatus(_ *cobra.Command, args []string) error {
	var ws *workspace.Workspace
	var err error
	if len(args) == 1 {
		ws, err = workspace.Resolve(args[0])
	} else if len(args) > 1 {
		return fmt.Errorf("too many arguments")
	} else {
		ws, err = workspace.Resolve(".")
	}
	if err != nil {
		return err
	}

	sessionsDir := filepath.Join(ws.Root, ".atlas", "sessions")
	id := statusSessionID

	if id == "" {
		latestID, err := session.GetLatestID(sessionsDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("no Atlas sessions found. The directory does not have an .atlas folder (project not deployed yet)")
			}
			return fmt.Errorf("could not find latest session: %w", err)
		}
		id = latestID
	}

	sess, err := session.Load(sessionsDir, id)
	if err != nil {
		return fmt.Errorf("could not load session %s: %w", id, err)
	}

	sessionDir := session.SessionDir(sessionsDir, id)

	planner, _ := orchestrator.LoadPlanner(sessionDir)
	buildState, _ := orchestrator.LoadBuild(sessionDir)
	deployState, _ := orchestrator.LoadDeployment(sessionDir)

	if statusJSON {
		output := map[string]interface{}{
			"session":    sess,
			"planner":    planner,
			"build":      buildState,
			"deployment": deployState,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	styleTitle := lipgloss.NewStyle().Bold(true).Foreground(cliutil.ColorInfo)
	styleSubtitle := lipgloss.NewStyle().Bold(true).Underline(true)
	styleValue := lipgloss.NewStyle().Foreground(cliutil.ColorHighlight)
	styleMuted := lipgloss.NewStyle().Foreground(cliutil.ColorDim)

	fmt.Println(styleTitle.Render("Atlas Status"))
	fmt.Printf("Session: %s\n", styleValue.Render(sess.ID))
	fmt.Printf("Status:  %s\n", statusColor(sess.Status).Render(strings.Title(sess.Status)))
	fmt.Printf("Started: %s\n", styleMuted.Render(timeSince(sess.CreatedAt)))
	fmt.Println()

	fmt.Println(styleSubtitle.Render("Pipeline Progress"))

	if planner != nil {
		styleSuccess := lipgloss.NewStyle().Foreground(cliutil.ColorSuccess)
		styleError := lipgloss.NewStyle().Foreground(cliutil.ColorError)
		styleHighlight := lipgloss.NewStyle().Foreground(cliutil.ColorHighlight)

		for _, step := range planner.Completed {
			fmt.Printf(" %s %s\n", styleSuccess.Render("✓"), stepLabel(step, planner, buildState))
		}
		if sess.Status == "running" && planner.CurrentStep != "" {
			fmt.Printf(" %s %s\n", styleHighlight.Render("⟳"), stepLabel(planner.CurrentStep, planner, buildState))
		}
		for _, step := range planner.Pending {
			fmt.Printf("   %s\n", styleMuted.Render(stepLabel(step, nil, nil)))
		}
		for _, step := range planner.Failed {
			fmt.Printf(" %s %s\n", styleError.Render("✗"), stepLabel(step, planner, buildState))
		}
	} else {
		fmt.Println("  (No pipeline state available)")
	}

	if deployState != nil {
		fmt.Println()
		fmt.Println(styleSubtitle.Render("Deployment"))
		fmt.Printf("  Provider: %s\n", deployState.Provider)
		if deployState.DeploymentURL != nil {
			fmt.Printf("  URL:      %s\n", styleValue.Render(*deployState.DeploymentURL))

			// Live HTTP GET health check
			liveStatus := "Unreachable"
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(*deployState.DeploymentURL)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					liveStatus = lipgloss.NewStyle().Foreground(cliutil.ColorSuccess).Render(fmt.Sprintf("Up (%d OK)", resp.StatusCode))
				} else {
					liveStatus = lipgloss.NewStyle().Foreground(cliutil.ColorError).Render(fmt.Sprintf("Down (%d)", resp.StatusCode))
				}
			} else {
				liveStatus = lipgloss.NewStyle().Foreground(cliutil.ColorError).Render("Unreachable")
			}
			fmt.Printf("  Health:   %s\n", liveStatus)
		} else {
			fmt.Printf("  Health:   %s\n", healthStatus(deployState))
		}
	}

	return nil
}
