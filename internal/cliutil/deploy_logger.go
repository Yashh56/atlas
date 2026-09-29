package cliutil

import (
	"fmt"
	"time"
)

type DeployLogger struct {
	jsonMode bool
	spinner  *Spinner
}

func NewDeployLogger(jsonMode bool) *DeployLogger {
	return &DeployLogger{jsonMode: jsonMode}
}

// Section starts a new major phase (e.g. "◆ Detect")
func (l *DeployLogger) Section(name string) {
	if l.jsonMode {
		return
	}
	fmt.Printf("\n%s\n", StylePrimary.Render("◆ "+name))
}

// StepLog prints a sub-step (e.g. "  ✓ Next.js")
func (l *DeployLogger) StepLog(status StepStatus, msg string, duration ...time.Duration) {
	if l.jsonMode {
		return
	}
	var icon string
	switch status {
	case StepPending:
		icon = StyleMuted.Render("○")
	case StepSuccess:
		icon = StyleSuccess.Render("✓")
	case StepFailed:
		icon = StyleError.Render("✗")
	case StepWarning:
		icon = StyleWarning.Render("⚠")
	case StepInfo:
		icon = StyleInfo.Render("→")
	case StepSkipped:
		icon = StyleMuted.Render("—")
	}

	durStr := ""
	if len(duration) > 0 && duration[0] > 0 {
		durStr = StyleMuted.Render(fmt.Sprintf("(%.1fs)", duration[0].Seconds()))
	}

	fmt.Printf("  %s %s %s\n", icon, StyleBody.Render(msg), durStr)
}

func (l *DeployLogger) StartSpinner(msg string) {
	if l.jsonMode || NonInteractive {
		fmt.Printf("  → %s...\n", msg)
		return
	}
	// Prefix with spaces to indent under the section
	l.spinner = StartSpinner("  " + msg)
}

func (l *DeployLogger) StopSpinner(status StepStatus, finalMsg string, duration ...time.Duration) {
	if l.spinner != nil {
		l.spinner.Stop()
		l.spinner = nil
	}
	if finalMsg != "" {
		l.StepLog(status, finalMsg, duration...)
	}
}

func (l *DeployLogger) Box(msg string) {
	if l.jsonMode {
		return
	}
	fmt.Printf("\n%s\n", FormatBox(msg))
}

func (l *DeployLogger) ErrorBox(err error, suggestion string) {
	if l.jsonMode {
		return
	}
	box := StyleBox.Copy().BorderForeground(ColorError).Render(fmt.Sprintf(
		"%s %s\n\n%s\n\n%s\n%s",
		IconError, StyleError.Bold(true).Render("Deployment failed"),
		err.Error(),
		StyleMuted.Render("Suggested actions"),
		StyleInfo.Render("→ ")+suggestion,
	))
	fmt.Printf("\n%s\n", box)
}

const StepWarning StepStatus = 99
const StepInfo StepStatus = 100
