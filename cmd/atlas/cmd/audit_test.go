package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/audit"
)

func TestAuditCmd_HumanOutput(t *testing.T) {
	tempDir := t.TempDir()
	envExample := filepath.Join(tempDir, ".env.example")
	_ = os.WriteFile(envExample, []byte("API_SECRET=\nDATABASE_URL=\n"), 0644)

	// Set one in .env
	_ = os.WriteFile(filepath.Join(tempDir, ".env"), []byte("API_SECRET=val\n"), 0644)

	auditProviderFlag = ""
	auditStrictFlag = false
	auditDirFlag = ""
	auditWorkspaceFlag = ""
	jsonFlag = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"audit", tempDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuditCmd_JSONOutput(t *testing.T) {
	tempDir := t.TempDir()
	envExample := filepath.Join(tempDir, ".env.example")
	_ = os.WriteFile(envExample, []byte("MISSING_SECRET=\n"), 0644)

	origJSON := jsonFlag
	origStrict := auditStrictFlag
	defer func() {
		jsonFlag = origJSON
		auditStrictFlag = origStrict
	}()

	jsonFlag = false
	auditStrictFlag = false
	auditProviderFlag = ""
	auditDirFlag = ""
	auditWorkspaceFlag = ""

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"audit", tempDir, "--json", "--strict"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected strict error on missing secrets, got nil")
	}
	if !strings.Contains(err.Error(), "secret audit failed: 1 missing secrets") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuditCmd_WorkspaceFlags(t *testing.T) {
	tempDir := t.TempDir()
	pnpmWs := filepath.Join(tempDir, "pnpm-workspace.yaml")
	_ = os.WriteFile(pnpmWs, []byte("packages:\n  - 'apps/*'\n"), 0644)

	webDir := filepath.Join(tempDir, "apps", "web")
	_ = os.MkdirAll(webDir, 0755)
	_ = os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name": "web"}`), 0644)
	_ = os.WriteFile(filepath.Join(webDir, ".env.example"), []byte("WEB_SECRET=\n"), 0644)

	jsonFlag = false
	auditStrictFlag = false
	auditDirFlag = ""
	auditWorkspaceFlag = ""

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"audit", tempDir, "--workspace", "web", "--json"})

	err := rootCmd.Execute()
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var captured bytes.Buffer
	_, _ = captured.ReadFrom(r)

	var report audit.SecretAuditReport
	if err := json.Unmarshal(captured.Bytes(), &report); err != nil {
		t.Fatalf("failed to unmarshal JSON report: %v, raw: %s", err, captured.String())
	}

	if report.WorkspaceRoot != webDir {
		t.Errorf("expected WorkspaceRoot %q, got %q", webDir, report.WorkspaceRoot)
	}
	if report.MissingCount != 1 {
		t.Errorf("expected 1 missing secret, got %d", report.MissingCount)
	}
}
