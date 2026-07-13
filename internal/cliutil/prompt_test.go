package cliutil_test

import (
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/cliutil"
)

func TestPromptSecretFromReader_EmptyInput(t *testing.T) {
	reader := strings.NewReader("\n")
	_, err := cliutil.PromptSecretFromReaderForTest("test", reader)
	if err == nil {
		t.Fatal("expected error on empty input, got nil")
	}
	if !strings.Contains(err.Error(), "empty input") {
		t.Errorf("expected 'empty input' error, got %v", err)
	}
}

func TestPromptSecretFromReader_ValidInput(t *testing.T) {
	reader := strings.NewReader("  mysecretkey  \n")
	key, err := cliutil.PromptSecretFromReaderForTest("test", reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "mysecretkey" {
		t.Errorf("expected trimmed key 'mysecretkey', got %q", key)
	}
}
