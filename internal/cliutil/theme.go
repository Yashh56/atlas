package cliutil

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	ColorSuccess   = lipgloss.Color("46")  // Bright Green
	ColorError     = lipgloss.Color("196") // Bright Red
	ColorWarning   = lipgloss.Color("214") // Orange/Yellow
	ColorInfo      = lipgloss.Color("39")  // Cyan/Blue
	ColorHighlight = lipgloss.Color("205") // Pink/Magenta
	ColorDim       = lipgloss.Color("240") // Dark Gray

	// Icons
	IconSuccess = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("✓")
	IconError   = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("✗")
	IconWarning = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("⚠")
	IconArrow   = lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("→")
	IconDot     = lipgloss.NewStyle().Foreground(ColorInfo).Render("•")

	// Text Styles
	StyleHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")). // Purple background
			Bold(true).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	StyleSubtext   = lipgloss.NewStyle().Foreground(ColorDim)
	StyleHighlight = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)
	StyleBold      = lipgloss.NewStyle().Bold(true)
	StylePrompt    = lipgloss.NewStyle().Foreground(ColorInfo).Bold(true)

	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(1, 4).
			MarginTop(1)
)

// FormatHeader renders a prominent header block.
func FormatHeader(title string) string {
	return StyleHeader.Render(strings.ToUpper(title))
}

// FormatStep renders a step with a highlighted value.
func FormatStep(label, value string) string {
	return fmt.Sprintf("%s %s", label, StyleHighlight.Render(value))
}

// FormatSuccess renders a success message with an aligned label.
func FormatSuccess(label, value string) string {
	// Pad the label to a fixed width (e.g., 14 chars)
	paddedLabel := fmt.Sprintf("%-12s", label)
	return fmt.Sprintf("%s %s %s", IconSuccess, StyleSubtext.Render(paddedLabel), StyleHighlight.Render(value))
}

// FormatBox renders a text string inside a beautifully styled box.
func FormatBox(text string) string {
	return StyleBox.Render(text)
}
