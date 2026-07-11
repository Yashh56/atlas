package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

func TestWriteFile_NormalWrite(t *testing.T) {
	ws := t.TempDir()
	tool := tools.WriteFile{
		WorkspaceRoot: ws,
		Path:          "test.txt",
		Content:       "hello world",
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}

	data, err := os.ReadFile(filepath.Join(ws, "test.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(data))
	}
	if res.Output != "Wrote 11 bytes to test.txt" {
		t.Errorf("unexpected output: %s", res.Output)
	}
}

func TestWriteFile_CreatesParentDirs(t *testing.T) {
	ws := t.TempDir()
	tool := tools.WriteFile{
		WorkspaceRoot: ws,
		Path:          "deep/nested/dir/test.txt",
		Content:       "nested",
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}

	data, err := os.ReadFile(filepath.Join(ws, "deep", "nested", "dir", "test.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "nested" {
		t.Errorf("expected 'nested', got '%s'", string(data))
	}
}

func TestWriteFile_PathTraversalRejection(t *testing.T) {
	ws := t.TempDir()
	tool := tools.WriteFile{
		WorkspaceRoot: ws,
		Path:          "../outside.txt",
		Content:       "hacked",
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected failure for path traversal, got success")
	}
	if res.Error == "" {
		t.Fatalf("expected error message, got empty string")
	}
}

func TestWriteFile_Overwrite(t *testing.T) {
	ws := t.TempDir()
	path := filepath.Join(ws, "test.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := tools.WriteFile{
		WorkspaceRoot: ws,
		Path:          "test.txt",
		Content:       "new",
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("expected 'new', got '%s'", string(data))
	}
}
