package cliutil

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"
)

// PromptSecret prompts the user for a secret using a masked input (no echo).
func PromptSecret(label string) (string, error) {
	var val string
	err := huh.NewInput().
		Title(label).
		EchoMode(huh.EchoModePassword).
		Value(&val).
		Run()
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", fmt.Errorf("empty input, nothing stored")
	}
	return val, nil
}

// PromptSecretFromReaderForTest is an exported variant for testing environments.
func PromptSecretFromReaderForTest(label string, r io.Reader) (string, error) {
	fmt.Printf("%s: ", label)
	bytes, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	key := strings.TrimSpace(string(bytes))
	if key == "" {
		return "", fmt.Errorf("empty input, nothing stored")
	}

	return key, nil
}

// PromptConfirm prompts the user for a y/N response and returns true if yes.
func PromptConfirm(label string) (bool, error) {
	var val bool
	err := huh.NewConfirm().
		Title(label).
		Value(&val).
		Run()
	if err != nil {
		return false, err
	}
	return val, nil
}
