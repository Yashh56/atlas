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

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("setting raw terminal: %w", err)
	}

	var pw []byte
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil && err != io.EOF {
			term.Restore(fd, oldState)
			return "", fmt.Errorf("reading input: %w", err)
		}
		if n == 0 {
			break
		}
		b := buf[0]
		if b == '\r' || b == '\n' {
			break
		}
		if b == 3 { // Ctrl+C
			term.Restore(fd, oldState)
			return "", fmt.Errorf("interrupted")
		}
		if b == 8 || b == 127 { // Backspace
			if len(pw) > 0 {
				pw = pw[:len(pw)-1]
				fmt.Print("\b \b")
			}
			continue
		}
		pw = append(pw, b)
		fmt.Print("*")
	}

	term.Restore(fd, oldState)
	fmt.Println()

	key := strings.TrimSpace(string(pw))
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

// PromptConfirm prompts the user for a y/N response and returns true if yes.
// It reads a single keystroke in raw mode, so the user doesn't need to hit Enter.
func PromptConfirm(label string) (bool, error) {
	fmt.Printf("%s: ", label)

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback for non-TTY or tests
		var ans string
		fmt.Scanln(&ans)
		s := strings.ToLower(strings.TrimSpace(ans))
		return s == "y" || s == "yes", nil
	}

	var b [1]byte
	for {
		n, err := os.Stdin.Read(b[:])
		if err != nil || n == 0 {
			term.Restore(fd, oldState)
			return false, err
		}
		if b[0] == 3 { // Ctrl+C
			term.Restore(fd, oldState)
			return false, fmt.Errorf("interrupted")
		}
		if b[0] == 'y' || b[0] == 'Y' {
			fmt.Print("y\r\n")
			term.Restore(fd, oldState)
			return true, nil
		}
		if b[0] == 'n' || b[0] == 'N' {
			fmt.Print("n\r\n")
			term.Restore(fd, oldState)
			return false, nil
		}
		if b[0] == '\r' || b[0] == '\n' {
			fmt.Print("\r\n")
			term.Restore(fd, oldState)
			return false, nil // default to No
		}
	}
}
