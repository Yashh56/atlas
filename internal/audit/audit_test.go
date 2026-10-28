package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTemplateKeys(t *testing.T) {
	tmpDir := t.TempDir()
	envExample := filepath.Join(tmpDir, ".env.example")
	content := `# Database config
DATABASE_URL=postgres://user:pass@localhost:5432/db
export JWT_SECRET=supersecret
STRIPE_KEY=
# Comment line

INVALID-KEY=test
ANOTHER_VAR=some_val
`
	if err := os.WriteFile(envExample, []byte(content), 0644); err != nil {
		t.Fatalf("writing .env.example: %v", err)
	}

	keys, err := parseTemplateKeys(envExample)
	if err != nil {
		t.Fatalf("parseTemplateKeys failed: %v", err)
	}

	expected := []string{"DATABASE_URL", "JWT_SECRET", "STRIPE_KEY", "ANOTHER_VAR"}
	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d: %v", len(expected), len(keys), keys)
	}

	for i, k := range expected {
		if keys[i] != k {
			t.Errorf("expected key[%d] to be %q, got %q", i, k, keys[i])
		}
	}
}

func TestScanSourceFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. JS / TS file
	tsFile := filepath.Join(tmpDir, "index.ts")
	tsContent := `
const apiKey = process.env.API_KEY;
const dbHost = process.env['DB_HOST'];
const nodeEnv = process.env.NODE_ENV;
`
	if err := os.WriteFile(tsFile, []byte(tsContent), 0644); err != nil {
		t.Fatalf("writing index.ts: %v", err)
	}
	tsKeys, err := scanSourceFile(tsFile)
	if err != nil {
		t.Fatalf("scanSourceFile(tsFile) failed: %v", err)
	}
	expectedTS := map[string]bool{"API_KEY": true, "DB_HOST": true, "NODE_ENV": true}
	for _, k := range tsKeys {
		if !expectedTS[k] {
			t.Errorf("unexpected key in TS: %s", k)
		}
	}
	if len(tsKeys) != len(expectedTS) {
		t.Errorf("expected %d keys in TS, got %d: %v", len(expectedTS), len(tsKeys), tsKeys)
	}

	// 2. Python file
	pyFile := filepath.Join(tmpDir, "app.py")
	pyContent := `
import os
db = os.environ.get("PY_DB_URL")
secret = os.environ['PY_SECRET']
db_url = os.environ["DATABASE_URL"]
mode = os.getenv("PY_MODE")
`
	if err := os.WriteFile(pyFile, []byte(pyContent), 0644); err != nil {
		t.Fatalf("writing app.py: %v", err)
	}
	pyKeys, err := scanSourceFile(pyFile)
	if err != nil {
		t.Fatalf("scanSourceFile(pyFile) failed: %v", err)
	}
	expectedPy := map[string]bool{"PY_DB_URL": true, "PY_SECRET": true, "DATABASE_URL": true, "PY_MODE": true}
	for _, k := range pyKeys {
		if !expectedPy[k] {
			t.Errorf("unexpected key in Py: %s", k)
		}
	}
	if len(pyKeys) != len(expectedPy) {
		t.Errorf("expected %d keys in Py, got %d: %v", len(expectedPy), len(pyKeys), pyKeys)
	}

	// 3. Go file
	goFile := filepath.Join(tmpDir, "main.go")
	goContent := `
package main
import "os"
func main() {
    val := os.Getenv("GO_API_KEY")
    val2, _ := os.LookupEnv("GO_SECRET")
}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatalf("writing main.go: %v", err)
	}
	goKeys, err := scanSourceFile(goFile)
	if err != nil {
		t.Fatalf("scanSourceFile(goFile) failed: %v", err)
	}
	expectedGo := map[string]bool{"GO_API_KEY": true, "GO_SECRET": true}
	for _, k := range goKeys {
		if !expectedGo[k] {
			t.Errorf("unexpected key in Go: %s", k)
		}
	}
	if len(goKeys) != len(expectedGo) {
		t.Errorf("expected %d keys in Go, got %d: %v", len(expectedGo), len(goKeys), goKeys)
	}
}

func TestAuditWorkspace_FullFlow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .env.example
	envExample := filepath.Join(tmpDir, ".env.example")
	_ = os.WriteFile(envExample, []byte("FOUND_IN_ENV=foo\nMISSING_VAR=\nNODE_ENV=production\nCONFIGURED_VAR=\n"), 0644)

	// Create src/index.js
	srcDir := filepath.Join(tmpDir, "src")
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "index.js"), []byte("const x = process.env.FOUND_IN_OS;\nconst p = process.env.PORT;\n"), 0644)

	// Create .env with FOUND_IN_ENV
	envFile := filepath.Join(tmpDir, ".env")
	_ = os.WriteFile(envFile, []byte("FOUND_IN_ENV=my-secret-val\n"), 0644)

	// Set OS env var
	t.Setenv("FOUND_IN_OS", "some-os-value")

	providerEnvKeys := []string{"CONFIGURED_VAR"}

	report, err := AuditWorkspace(tmpDir, "vercel", providerEnvKeys)
	if err != nil {
		t.Fatalf("AuditWorkspace failed: %v", err)
	}

	if report.WorkspaceRoot != tmpDir {
		t.Errorf("expected WorkspaceRoot %q, got %q", tmpDir, report.WorkspaceRoot)
	}
	if report.Provider != "vercel" {
		t.Errorf("expected Provider %q, got %q", "vercel", report.Provider)
	}

	// Verify NODE_ENV and PORT were excluded
	for _, s := range report.Secrets {
		if s.Key == "NODE_ENV" || s.Key == "PORT" {
			t.Errorf("standard system variable %s should be excluded", s.Key)
		}
	}

	// Verify counts:
	// CONFIGURED_VAR -> configured
	// FOUND_IN_ENV   -> found_local
	// FOUND_IN_OS    -> found_local
	// MISSING_VAR    -> missing
	if report.MissingCount != 1 {
		t.Errorf("expected 1 missing secret, got %d", report.MissingCount)
	}
	if report.FoundCount != 3 {
		t.Errorf("expected 3 found secrets, got %d", report.FoundCount)
	}

	statusMap := make(map[string]SecretStatus)
	for _, s := range report.Secrets {
		statusMap[s.Key] = s.Status
	}

	if statusMap["CONFIGURED_VAR"] != StatusConfigured {
		t.Errorf("expected CONFIGURED_VAR to be %v, got %v", StatusConfigured, statusMap["CONFIGURED_VAR"])
	}
	if statusMap["FOUND_IN_ENV"] != StatusFoundLocal {
		t.Errorf("expected FOUND_IN_ENV to be %v, got %v", StatusFoundLocal, statusMap["FOUND_IN_ENV"])
	}
	if statusMap["FOUND_IN_OS"] != StatusFoundLocal {
		t.Errorf("expected FOUND_IN_OS to be %v, got %v", StatusFoundLocal, statusMap["FOUND_IN_OS"])
	}
	if statusMap["MISSING_VAR"] != StatusMissing {
		t.Errorf("expected MISSING_VAR to be %v, got %v", StatusMissing, statusMap["MISSING_VAR"])
	}

	// Verify deterministic sorting
	for i := 1; i < len(report.Secrets); i++ {
		if report.Secrets[i-1].Key >= report.Secrets[i].Key {
			t.Errorf("secrets not sorted alphabetically: %s >= %s", report.Secrets[i-1].Key, report.Secrets[i].Key)
		}
	}
}

func TestAuditWorkspace_LocalEnvFiles(t *testing.T) {
	tmpDir := t.TempDir()

	envExample := filepath.Join(tmpDir, ".env.example")
	_ = os.WriteFile(envExample, []byte("VAR_LOCAL=\nVAR_PROD=\nVAR_PROD_LOCAL=\n"), 0644)

	_ = os.WriteFile(filepath.Join(tmpDir, ".env.local"), []byte("VAR_LOCAL=1\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, ".env.production"), []byte("VAR_PROD=1\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, ".env.production.local"), []byte("VAR_PROD_LOCAL=1\n"), 0644)

	report, err := AuditWorkspace(tmpDir, "", nil)
	if err != nil {
		t.Fatalf("AuditWorkspace failed: %v", err)
	}

	if report.MissingCount != 0 {
		t.Errorf("expected 0 missing, got %d", report.MissingCount)
	}
	if report.FoundCount != 3 {
		t.Errorf("expected 3 found, got %d", report.FoundCount)
	}
}

func TestAuditWorkspace_EmptyEnvValue(t *testing.T) {
	tmpDir := t.TempDir()

	envExample := filepath.Join(tmpDir, ".env.example")
	_ = os.WriteFile(envExample, []byte("SET_VAR=\nEMPTY_QUOTED=\nEMPTY_BARE=\n"), 0644)

	envFile := filepath.Join(tmpDir, ".env")
	_ = os.WriteFile(envFile, []byte("SET_VAR=secret_value\nEMPTY_QUOTED=\"\"\nEMPTY_BARE=\n"), 0644)

	report, err := AuditWorkspace(tmpDir, "", nil)
	if err != nil {
		t.Fatalf("AuditWorkspace failed: %v", err)
	}

	statusMap := make(map[string]SecretStatus)
	for _, s := range report.Secrets {
		statusMap[s.Key] = s.Status
	}

	if statusMap["SET_VAR"] != StatusFoundLocal {
		t.Errorf("expected SET_VAR to be %v, got %v", StatusFoundLocal, statusMap["SET_VAR"])
	}
	if statusMap["EMPTY_QUOTED"] != StatusMissing {
		t.Errorf("expected EMPTY_QUOTED to be %v, got %v", StatusMissing, statusMap["EMPTY_QUOTED"])
	}
	if statusMap["EMPTY_BARE"] != StatusMissing {
		t.Errorf("expected EMPTY_BARE to be %v, got %v", StatusMissing, statusMap["EMPTY_BARE"])
	}
	if report.MissingCount != 2 {
		t.Errorf("expected 2 missing keys, got %d", report.MissingCount)
	}
}

func TestAuditWorkspace_TemplateError(t *testing.T) {
	tmpDir := t.TempDir()

	// Make .env.example a directory so reading it fails with an error (not IsNotExist)
	envExample := filepath.Join(tmpDir, ".env.example")
	if err := os.Mkdir(envExample, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := AuditWorkspace(tmpDir, "", nil)
	if err == nil {
		t.Fatal("expected error when .env.example is an unreadable directory, got nil")
	}
}

