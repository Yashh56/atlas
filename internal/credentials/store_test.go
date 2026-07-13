package credentials_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	keyring "github.com/zalando/go-keyring"

	"github.com/Yashh56/atlas/internal/credentials"
)

// TestMain probes the real keychain once; skips secret tests if unavailable.
var keychainAvailable bool

func TestMain(m *testing.M) {
	// Try a harmless Set/Get/Delete to see if the keychain works.
	probe := "atlas-test-probe"
	err := keyring.Set("atlas-probe", probe, "ok")
	if err == nil {
		_ = keyring.Delete("atlas-probe", probe)
		keychainAvailable = true
	}
	os.Exit(m.Run())
}

// openTemp creates a Store backed by a temp dir (never touches the real user config dir).
func openTemp(t *testing.T) *credentials.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := credentials.OpenWithDir(dir)
	if err != nil {
		t.Fatalf("OpenWithDir: %v", err)
	}
	return s
}

func TestMetaRoundTrip(t *testing.T) {
	s := openTemp(t)
	now := time.Now().Truncate(time.Second)

	cred := credentials.ProviderCredential{
		Provider:   "vercel",
		Method:     credentials.MethodCLISession,
		VerifiedAt: now,
		Account:    "you@example.com",
	}
	if err := s.SetMeta(cred); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}

	got, ok, err := s.GetMeta("vercel")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !ok {
		t.Fatal("GetMeta: expected to find vercel, got not found")
	}
	if got.Provider != cred.Provider {
		t.Errorf("Provider: want %q got %q", cred.Provider, got.Provider)
	}
	if got.Method != cred.Method {
		t.Errorf("Method: want %q got %q", cred.Method, got.Method)
	}
	if got.Account != cred.Account {
		t.Errorf("Account: want %q got %q", cred.Account, got.Account)
	}
	if !got.VerifiedAt.Equal(now) {
		t.Errorf("VerifiedAt: want %v got %v", now, got.VerifiedAt)
	}
}

func TestMetaMissing(t *testing.T) {
	s := openTemp(t)
	_, ok, err := s.GetMeta("nonexistent")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if ok {
		t.Fatal("expected not found, got found")
	}
}

func TestMetaMultipleProviders(t *testing.T) {
	s := openTemp(t)
	now := time.Now().Truncate(time.Second)

	for _, name := range []string{"vercel", "render", "fly"} {
		if err := s.SetMeta(credentials.ProviderCredential{
			Provider:   name,
			Method:     credentials.MethodEnvVar,
			VerifiedAt: now,
		}); err != nil {
			t.Fatalf("SetMeta(%s): %v", name, err)
		}
	}

	for _, name := range []string{"vercel", "render", "fly"} {
		got, ok, err := s.GetMeta(name)
		if err != nil || !ok {
			t.Fatalf("GetMeta(%s): err=%v ok=%v", name, err, ok)
		}
		if got.Provider != name {
			t.Errorf("wanted %q got %q", name, got.Provider)
		}
	}
}

// credentialsJSON is never written to a temp file that has a test secret in it.
func TestCredentialsJSONHasNoSecretField(t *testing.T) {
	s := openTemp(t)
	now := time.Now().Truncate(time.Second)
	if err := s.SetMeta(credentials.ProviderCredential{
		Provider:   "vercel",
		Method:     credentials.MethodStoredToken,
		VerifiedAt: now,
	}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	// Read the raw JSON and make sure "token" or "secret" don't appear as field keys.
	path := filepath.Join(s.Dir(), "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading credentials.json: %v", err)
	}
	for _, forbidden := range []string{`"token"`, `"secret"`, `"password"`, `"key"`} {
		if contains(string(data), forbidden) {
			t.Errorf("credentials.json contains forbidden field %s: %s", forbidden, data)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}())
}

func TestSecretOps(t *testing.T) {
	if !keychainAvailable {
		t.Skip("no keychain backend available — skipping secret ops test")
	}
	s := openTemp(t)

	if err := s.SetSecret("vercel", "test-token-value"); err != nil {
		t.Fatalf("SetSecret: %v", err)
	}
	got, err := s.GetSecret("vercel")
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if got != "test-token-value" {
		t.Errorf("GetSecret: want %q got %q", "test-token-value", got)
	}
	if err := s.DeleteSecret("vercel"); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	_, err = s.GetSecret("vercel")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestFallbackSecretOps(t *testing.T) {
	// Force fallback by using a mock that returns ErrUnsupportedPlatform.
	// Since we can't easily inject the keyring in the current design without
	// test-specific build tags, we instead test the exported fallback helpers
	// indirectly by calling SetSecret on a store with a dir that works,
	// then checking the fallback file exists and contains the expected value
	// when keychain is not available.
	// This test is always run (doesn't require keychain).
	s := openTemp(t)
	// Directly call the exported fallback helpers via SetFallbackSecret (visible only in tests).
	// We verify the fallback file stays 0600.
	fallbackPath := filepath.Join(s.Dir(), "secrets.fallback.json")

	// We can verify the fallback file permissions only if it gets created.
	// Write via the exported test-helper method.
	if err := s.SetFallbackSecretForTest("render", "render-token"); err != nil {
		t.Fatalf("SetFallbackSecretForTest: %v", err)
	}
	info, err := os.Stat(fallbackPath)
	if err != nil {
		t.Fatalf("stat fallback file: %v", err)
	}
	// Windows does not honour Unix permission bits: os.WriteFile(0600) results in 0666.
	// Only check the restriction on non-Windows platforms.
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("fallback file perms: want 0600 got %04o", perm)
		}
	} else {
		_ = info // satisfy "declared and not used" in case other checks are added
	}
	val, err := s.GetFallbackSecretForTest("render")
	if err != nil {
		t.Fatalf("GetFallbackSecretForTest: %v", err)
	}
	if val != "render-token" {
		t.Errorf("want %q got %q", "render-token", val)
	}
}
