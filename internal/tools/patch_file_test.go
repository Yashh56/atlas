package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/tools"
)

func TestPatchFile_Execute_Success(t *testing.T) {
	ws := t.TempDir()
	filePath := filepath.Join(ws, "test.txt")
	originalContent := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(filePath, []byte(originalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	pf := tools.PatchFile{
		WorkspaceRoot: ws,
		Path:          "test.txt",
		OldStr:        "line 2\n",
		NewStr:        "LINE TWO\n",
	}
	res, err := pf.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got failure: %v", res.Error)
	}

	b, _ := os.ReadFile(filePath)
	if string(b) != "line 1\nLINE TWO\nline 3\n" {
		t.Fatalf("unexpected content: %q", string(b))
	}
}

func TestPatchFile_Execute_CRLF_Match(t *testing.T) {
	ws := t.TempDir()
	filePath := filepath.Join(ws, "test.txt")
	// File has CRLF line endings
	originalContent := "line 1\r\nline 2\r\nline 3\r\n"
	if err := os.WriteFile(filePath, []byte(originalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	pf := tools.PatchFile{
		WorkspaceRoot: ws,
		Path:          "test.txt",
		// LLM generates LF strings typically
		OldStr: "line 2\n",
		NewStr: "LINE TWO\n",
	}
	res, err := pf.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got failure: %v", res.Error)
	}

	b, _ := os.ReadFile(filePath)
	expectedContent := "line 1\r\nLINE TWO\r\nline 3\r\n"
	if string(b) != expectedContent {
		t.Fatalf("unexpected content: %q", string(b))
	}
}

func TestPatchFile_Execute_NoMatch(t *testing.T) {
	ws := t.TempDir()
	filePath := filepath.Join(ws, "test.txt")
	os.WriteFile(filePath, []byte("hello world"), 0o644)

	pf := tools.PatchFile{
		WorkspaceRoot: ws,
		Path:          "test.txt",
		OldStr:        "missing",
		NewStr:        "found",
	}
	res, _ := pf.Execute(context.Background(), nil)
	if res.Success {
		t.Fatal("expected failure on no match")
	}
	if !strings.Contains(res.Error, "old_str not found in file") {
		t.Fatalf("unexpected error message: %q", res.Error)
	}
}

func TestPatchFile_Execute_MultipleMatch(t *testing.T) {
	ws := t.TempDir()
	filePath := filepath.Join(ws, "test.txt")
	os.WriteFile(filePath, []byte("dup\ndup\n"), 0o644)

	pf := tools.PatchFile{
		WorkspaceRoot: ws,
		Path:          "test.txt",
		OldStr:        "dup\n",
		NewStr:        "fix\n",
	}
	res, _ := pf.Execute(context.Background(), nil)
	if res.Success {
		t.Fatal("expected failure on multiple match")
	}
	if !strings.Contains(res.Error, "must be unique") {
		t.Fatalf("unexpected error message: %q", res.Error)
	}
}

func TestPatchFile_Execute_PathEscape(t *testing.T) {
	ws := t.TempDir()
	pf := tools.PatchFile{
		WorkspaceRoot: ws,
		Path:          "../outside.txt",
		OldStr:        "a",
		NewStr:        "b",
	}
	res, _ := pf.Execute(context.Background(), nil)
	if res.Success {
		t.Fatal("expected failure on path escape")
	}
}
