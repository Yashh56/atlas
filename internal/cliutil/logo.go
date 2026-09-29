package cliutil

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const atlasLogo = "\n" +
	"    ▲    \n" +
	"   ▲ ▲   \n" +
	"  ▲▲▲▲▲  \n" +
	" ▲     ▲ \n" +
	"▲       ▲"

// PrintWelcome prints a side-by-side logo and description.
func PrintWelcome() {
	logo := lipgloss.NewStyle().
		Foreground(DefaultTheme.Primary).
		Bold(true).
		Render(strings.TrimPrefix(atlasLogo, "\n"))

	title := StyleTitle.Render("Atlas")
	subtitle := StyleHighlight.Render("Autonomous deployment agent")
	desc := StyleBody.Render("Atlas is a CLI tool that autonomously deploys your projects.")
	helpMsg := StyleMuted.Render("Run 'atlas --help' to see available commands.")

	text := lipgloss.JoinVertical(lipgloss.Left,
		title+" — "+subtitle,
		"",
		desc,
		"",
		helpMsg,
	)

	text = lipgloss.NewStyle().PaddingLeft(4).Render(text)
	out := lipgloss.JoinHorizontal(lipgloss.Top, logo, text)

	fmt.Println("\n" + out + "\n")
}
