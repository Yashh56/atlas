package cliutil

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Success    lipgloss.Color
	Error      lipgloss.Color
	Warning    lipgloss.Color
	Info       lipgloss.Color
	Muted      lipgloss.Color
	Text       lipgloss.Color
	Background lipgloss.Color
	Border     lipgloss.Color
}

// AtlasTheme defines the standard color palette for the CLI.
// Indigo/Cyan branding.
var DefaultTheme = Theme{
	Primary:    lipgloss.Color("#6366F1"), // Indigo
	Secondary:  lipgloss.Color("#22D3EE"), // Cyan
	Success:    lipgloss.Color("#22C55E"), // Green
	Error:      lipgloss.Color("#EF4444"), // Red
	Warning:    lipgloss.Color("#F59E0B"), // Amber
	Info:       lipgloss.Color("#3B82F6"), // Blue
	Muted:      lipgloss.Color("#71717A"), // Gray 500
	Text:       lipgloss.Color("#FAFAFA"), // Almost White
	Background: lipgloss.Color("#18181B"), // Gray 900
	Border:     lipgloss.Color("#27272A"), // Gray 800
}
