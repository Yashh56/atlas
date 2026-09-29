package cliutil

import (
	"fmt"
	"strings"
)

// Success prints a standardized success message
func Success(msg string) {
	fmt.Printf("%s %s\n", IconSuccess, StyleBody.Render(msg))
}

// Error prints a standardized error message
func Error(msg string) {
	fmt.Printf("%s %s\n", IconError, StyleBody.Render(msg))
}

// Warning prints a standardized warning message
func Warning(msg string) {
	fmt.Printf("%s %s\n", IconWarning, StyleBody.Render(msg))
}

// Info prints a standardized info message
func Info(msg string) {
	fmt.Printf("%s %s\n", IconInfo, StyleBody.Render(msg))
}

// FormatHeader renders a prominent header block. (Kept for compatibility)
func FormatHeader(title string) string {
	return StyleHeader.Render(strings.ToUpper(title))
}

// FormatStep renders a step with a highlighted value. (Kept for compatibility)
func FormatStep(label, value string) string {
	return fmt.Sprintf("%s %s", label, StyleHighlight.Render(value))
}

// FormatSuccess renders a success message with an aligned label. (Kept for compatibility)
func FormatSuccess(label, value string) string {
	paddedLabel := fmt.Sprintf("%-12s", label)
	return fmt.Sprintf("%s %s %s", IconSuccess, StyleSubtext.Render(paddedLabel), StyleHighlight.Render(value))
}

// FormatBox renders a text string inside a beautifully styled box.
func FormatBox(text string) string {
	return StyleBox.Render(text)
}
