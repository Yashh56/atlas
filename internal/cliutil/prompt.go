package cliutil

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// PromptSecret prompts the user for a secret using a masked input (no echo).
func PromptSecret(label string) (string, error) {
	fmt.Printf("%s: ", label)
	// ReadPassword reads from the file descriptor without echoing input.
	bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // emit the newline the user's Enter keypress didn't produce
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	key := strings.TrimSpace(string(bytes))
	if key == "" {
		return "", fmt.Errorf("empty input, nothing stored")
	}

	return key, nil
}

// PromptSecretFromReaderForTest is an exported variant for testing environments where
// we can't capture a real TTY file descriptor and want to mock via an io.Reader.
func PromptSecretFromReaderForTest(label string, r io.Reader) (string, error) {
	fmt.Printf("%s: ", label)
	// We just read normally instead of masking, for the sake of the test.
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
