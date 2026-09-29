package cliutil

import "github.com/charmbracelet/lipgloss"

var (
	// Basic typography
	StyleTitle    = lipgloss.NewStyle().Foreground(DefaultTheme.Primary).Bold(true)
	StyleSubtitle = lipgloss.NewStyle().Foreground(DefaultTheme.Secondary)
	StyleBody     = lipgloss.NewStyle().Foreground(DefaultTheme.Text)
	StyleMuted    = lipgloss.NewStyle().Foreground(DefaultTheme.Muted)
	StyleLabel    = lipgloss.NewStyle().Foreground(DefaultTheme.Text).Bold(true)
	StyleValue    = lipgloss.NewStyle().Foreground(DefaultTheme.Secondary)

	// Status semantics
	StyleSuccess = lipgloss.NewStyle().Foreground(DefaultTheme.Success)
	StyleError   = lipgloss.NewStyle().Foreground(DefaultTheme.Error)
	StyleWarning = lipgloss.NewStyle().Foreground(DefaultTheme.Warning)
	StyleInfo    = lipgloss.NewStyle().Foreground(DefaultTheme.Info)

	// Specialty
	StyleCode      = lipgloss.NewStyle().Foreground(DefaultTheme.Secondary).Background(DefaultTheme.Border).Padding(0, 1)
	StyleCommand   = lipgloss.NewStyle().Foreground(DefaultTheme.Primary).Bold(true)
	StyleURL       = lipgloss.NewStyle().Foreground(DefaultTheme.Secondary).Underline(true)
	StyleHighlight = lipgloss.NewStyle().Foreground(DefaultTheme.Secondary).Bold(true)

	// Containers
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DefaultTheme.Border).
			Padding(1, 2)
)

// Back-compatibility aliases
var (
	ColorSuccess   = DefaultTheme.Success
	ColorError     = DefaultTheme.Error
	ColorWarning   = DefaultTheme.Warning
	ColorInfo      = DefaultTheme.Info
	ColorHighlight = DefaultTheme.Secondary
	ColorDim       = DefaultTheme.Muted

	StyleHeader  = StyleTitle.Padding(0, 1).MarginTop(1).MarginBottom(1).Background(lipgloss.Color("62")) // Kept temporarily
	StyleSubtext = StyleMuted
	StyleBold    = lipgloss.NewStyle().Bold(true)
	StylePrompt  = StyleInfo.Bold(true)
)
