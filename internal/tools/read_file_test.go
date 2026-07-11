package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

func TestReadFile(t *testing.T) {
	dir := t.TempDir()

	// Write a known file.
	content := "hello, atlas\nsecond line\n"
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name        string
		path        string
		wantSuccess bool
		wantOutput  string
	}{
		{
			name:        "existing file",
			path:        path,
			wantSuccess: true,
			wantOutput:  content,
		},
		{
			name:        "nonexistent file",
			path:        filepath.Join(dir, "missing.txt"),
			wantSuccess: false,
		},
	}

	stub := &session.Session{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tool := tools.ReadFile{Path: tc.path}
			result, err := tool.Execute(context.Background(), stub)
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if result.Success != tc.wantSuccess {
				t.Errorf("Success = %v, want %v; Error = %q", result.Success, tc.wantSuccess, result.Error)
			}
			if tc.wantSuccess && result.Output != tc.wantOutput {
				t.Errorf("Output mismatch:\ngot:  %q\nwant: %q", result.Output, tc.wantOutput)
			}
		})
	}
}

func TestReadFile_TooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.bin")

	// Create a 2 MB file (above the 1 MB cap).
	data := make([]byte, 2*1024*1024)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := tools.ReadFile{Path: path}
	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for oversized file")
	}
	if !strings.Contains(result.Error, "exceeds 1 MB limit") {
		t.Errorf("expected limit error, got: %q", result.Error)
	}
}
