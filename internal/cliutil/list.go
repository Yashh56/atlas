package cliutil

import (
	"github.com/charmbracelet/bubbles/list"
)

// ApplyListTheme applies the Atlas theme to a bubbles list delegate.
func ApplyListTheme(d *list.DefaultDelegate) {
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(DefaultTheme.Primary).
		BorderLeftForeground(DefaultTheme.Primary)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(DefaultTheme.Primary).
		BorderLeftForeground(DefaultTheme.Primary)
}
