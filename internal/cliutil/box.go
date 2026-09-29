package cliutil

// Box returns a beautifully styled box containing the provided text.
func Box(text string) string {
	return StyleBox.Render(text)
}

// Panel is an alias for Box for larger content areas.
func Panel(title, text string) string {
	boxStyle := StyleBox.Copy().
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	if title != "" {
		title = StyleTitle.Render(" " + title + " ")
		return boxStyle.Render(title + "\n\n" + text)
	}

	return boxStyle.Render(text)
}
