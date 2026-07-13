// Package credentials provides a global, per-user credential store for Atlas.
// Metadata (non-secret) is persisted to credentials.json in the store dir.
// Actual secrets are stored in the OS keychain via go-keyring, with a 0600
// file fallback when no keychain backend is available.
package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	keyring "github.com/zalando/go-keyring"
)

const (
	keyringSvc    = "atlas"
	metaFileName  = "credentials.json"
	fallbackFile  = "secrets.fallback.json"
	storeDirPerms = 0700
	filePerms     = 0600
)

// AuthMethod describes how a provider credential is held.
type AuthMethod string

const (
	MethodEnvVar      AuthMethod = "env_var"      // token already in shell env
	MethodStoredToken AuthMethod = "stored_token"  // Atlas holds it in the OS keychain
	MethodCLISession  AuthMethod = "cli_session"   // delegated to the provider's own CLI
)

// ProviderCredential holds public (non-secret) metadata about a provider's auth.
// Secret values are NEVER stored in this struct — they live only in the OS keychain
// or the fallback file, accessed through Store.GetSecret / Store.SetSecret.
type ProviderCredential struct {
	Provider   string     `json:"provider"`
	Method     AuthMethod `json:"method"`
	VerifiedAt time.Time  `json:"verified_at"`
	Account    string     `json:"account,omitempty"` // e.g. email from `vercel whoami`
}

// Store is a handle to the Atlas global credential directory.
type Store struct {
	dir             string
	fallbackWarned  bool
	fallbackWarnMu  sync.Mutex
}

// Open resolves os.UserConfigDir()/atlas, creates it if missing (mode 0700),
// and returns a ready-to-use Store.
func Open() (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("credentials: cannot determine user config dir: %w", err)
	}
	dir := filepath.Join(base, "atlas")
	if err := os.MkdirAll(dir, storeDirPerms); err != nil {
		return nil, fmt.Errorf("credentials: creating store dir %s: %w", dir, err)
	}
	return &Store{dir: dir}, nil
}

// OpenWithDir creates a Store backed by the given directory (used in tests to avoid
// touching the real user config dir). Creates the directory if it doesn't exist.
func OpenWithDir(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, storeDirPerms); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Dir returns the store directory path (used in tests to locate files).
func (s *Store) Dir() string { return s.dir }

// SetFallbackSecretForTest writes directly to the fallback file — exported only for testing.
func (s *Store) SetFallbackSecretForTest(provider, secret string) error {
	return s.setFallbackSecret(provider, secret)
}

// GetFallbackSecretForTest reads directly from the fallback file — exported only for testing.
func (s *Store) GetFallbackSecretForTest(provider string) (string, error) {
	return s.getFallbackSecret(provider)
}

// metaPath returns the path to the non-secret metadata JSON file.
func (s *Store) metaPath() string { return filepath.Join(s.dir, metaFileName) }

// fallbackPath returns the path to the encrypted-by-perms secrets fallback file.
func (s *Store) fallbackPath() string { return filepath.Join(s.dir, fallbackFile) }

// --- Metadata (non-secret) ---

// GetMeta retrieves the stored metadata for a provider.
// Returns (cred, true, nil) when found; (nil, false, nil) when not present.
func (s *Store) GetMeta(provider string) (*ProviderCredential, bool, error) {
	all, err := s.readAllMeta()
	if err != nil {
		return nil, false, err
	}
	cred, ok := all[provider]
	if !ok {
		return nil, false, nil
	}
	return &cred, true, nil
}

// SetMeta upserts the metadata for a provider.
func (s *Store) SetMeta(cred ProviderCredential) error {
	all, err := s.readAllMeta()
	if err != nil {
		return err
	}
	all[cred.Provider] = cred
	return s.writeAllMeta(all)
}

func (s *Store) readAllMeta() (map[string]ProviderCredential, error) {
	data, err := os.ReadFile(s.metaPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]ProviderCredential{}, nil
		}
		return nil, fmt.Errorf("credentials: reading metadata: %w", err)
	}
	var all map[string]ProviderCredential
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("credentials: parsing metadata: %w", err)
	}
	return all, nil
}

func (s *Store) writeAllMeta(all map[string]ProviderCredential) error {
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return fmt.Errorf("credentials: marshalling metadata: %w", err)
	}
	if err := os.WriteFile(s.metaPath(), data, filePerms); err != nil {
		return fmt.Errorf("credentials: writing metadata: %w", err)
	}
	return nil
}

// --- Secrets (via OS keychain, with fallback) ---

// SetSecret stores a secret for a provider in the OS keychain.
// Falls back to a 0600 file if the keychain is unavailable, printing a warning.
func (s *Store) SetSecret(provider, secret string) error {
	err := keyring.Set(keyringSvc, provider, secret)
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrUnsupportedPlatform) || isKeychainUnavailable(err) {
		s.warnFallback()
		return s.setFallbackSecret(provider, secret)
	}
	return fmt.Errorf("credentials: keychain set(%s): %w", provider, err)
}

// GetSecret retrieves a secret for a provider from the OS keychain (or fallback).
func (s *Store) GetSecret(provider string) (string, error) {
	val, err := keyring.Get(keyringSvc, provider)
	if err == nil {
		return val, nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		// Check fallback
		val2, ferr := s.getFallbackSecret(provider)
		if ferr != nil {
			return "", keyring.ErrNotFound
		}
		return val2, nil
	}
	if errors.Is(err, keyring.ErrUnsupportedPlatform) || isKeychainUnavailable(err) {
		s.warnFallback()
		return s.getFallbackSecret(provider)
	}
	return "", fmt.Errorf("credentials: keychain get(%s): %w", provider, err)
}

// DeleteSecret removes a secret for a provider from the OS keychain (or fallback).
func (s *Store) DeleteSecret(provider string) error {
	err := keyring.Delete(keyringSvc, provider)
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		// Also clean fallback in case it was stored there previously
		_ = s.deleteFallbackSecret(provider)
		return nil
	}
	if errors.Is(err, keyring.ErrUnsupportedPlatform) || isKeychainUnavailable(err) {
		s.warnFallback()
		return s.deleteFallbackSecret(provider)
	}
	return fmt.Errorf("credentials: keychain delete(%s): %w", provider, err)
}

// warnFallback prints a one-time security-posture warning when the fallback activates.
func (s *Store) warnFallback() {
	s.fallbackWarnMu.Lock()
	defer s.fallbackWarnMu.Unlock()
	if s.fallbackWarned {
		return
	}
	s.fallbackWarned = true
	fmt.Fprintf(os.Stderr,
		"\n⚠  WARNING: No OS keychain backend available. Atlas is falling back to a\n"+
			"   plaintext secrets file at %s (permissions 0600).\n"+
			"   This is a security posture downgrade — the OS keychain is strongly preferred.\n"+
			"   Install a keychain backend (e.g. gnome-keyring / kwallet on Linux) to suppress this.\n\n",
		s.fallbackPath(),
	)
}

// isKeychainUnavailable heuristically detects common keychain backend errors
// (e.g. D-Bus not running on headless Linux) that should trigger fallback.
func isKeychainUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, s := range []string{
		"dbus",
		"no session bus",
		"secret service",
		"could not connect",
		"keychain",
	} {
		if containsFold(msg, s) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				match := true
				for j := 0; j < len(sub); j++ {
					c1 := s[i+j]
					c2 := sub[j]
					if c1 >= 'A' && c1 <= 'Z' {
						c1 += 32
					}
					if c2 >= 'A' && c2 <= 'Z' {
						c2 += 32
					}
					if c1 != c2 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
			return false
		}())
}

// --- Fallback file helpers ---

func (s *Store) readFallbackAll() (map[string]string, error) {
	data, err := os.ReadFile(s.fallbackPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("credentials: reading fallback secrets file: %w", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("credentials: parsing fallback secrets file: %w", err)
	}
	return m, nil
}

func (s *Store) writeFallbackAll(m map[string]string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.fallbackPath(), data, filePerms)
}

func (s *Store) setFallbackSecret(provider, secret string) error {
	m, err := s.readFallbackAll()
	if err != nil {
		return err
	}
	m[provider] = secret
	return s.writeFallbackAll(m)
}

func (s *Store) getFallbackSecret(provider string) (string, error) {
	m, err := s.readFallbackAll()
	if err != nil {
		return "", err
	}
	val, ok := m[provider]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return val, nil
}

func (s *Store) deleteFallbackSecret(provider string) error {
	m, err := s.readFallbackAll()
	if err != nil {
		return err
	}
	delete(m, provider)
	return s.writeFallbackAll(m)
}
