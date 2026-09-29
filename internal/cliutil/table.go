package cliutil

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table renders a simple aligned key-value table.
func Table(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	maxLeft := 0
	for _, row := range rows {
		if len(row) > 0 {
			w := lipgloss.Width(row[0])
			if w > maxLeft {
				maxLeft = w
			}
		}
	}

	var out strings.Builder
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		left := row[0]
		pad := maxLeft - lipgloss.Width(left)
		if pad < 0 {
			pad = 0
		}

		out.WriteString(StyleMuted.Render(left))
		out.WriteString(strings.Repeat(" ", pad+2))

		if len(row) > 1 {
			out.WriteString(StyleBody.Render(row[1]))
		}
		out.WriteString("\n")
	}

	return out.String()
}
