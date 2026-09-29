package cliutil

import (
	"fmt"
	"os"
)

type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepSuccess
	StepFailed
	StepSkipped
)

func RenderStep(name string, status StepStatus, message string) string {
	var icon string
	switch status {
	case StepPending:
		icon = StyleMuted.Render("○")
	case StepRunning:
		icon = StylePrimary.Render("◆")
	case StepSuccess:
		icon = StyleSuccess.Render("✓")
	case StepFailed:
		icon = StyleError.Render("✗")
	case StepSkipped:
		icon = StyleMuted.Render("—")
	}

	if message != "" {
		message = " " + StyleMuted.Render(message)
	}

	return fmt.Sprintf("%s %s%s\n", icon, StyleBody.Render(name), message)
}

// Global flag to disable interactive UI in CI/JSON
var NonInteractive bool

func init() {
	if os.Getenv("CI") != "" || os.Getenv("NON_INTERACTIVE") != "" || os.Getenv("ATLAS_JSON") != "" {
		NonInteractive = true
	}
}
