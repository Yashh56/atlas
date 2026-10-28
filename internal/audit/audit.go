// Package audit inspects a workspace for environment variables and secrets
// required by templates and source code, verifying their presence without
// logging, saving, or exposing secret values.
package audit

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Yashh56/atlas/internal/workspace"
)

type SecretStatus string

const (
	StatusFoundLocal SecretStatus = "found_local"
	StatusConfigured SecretStatus = "configured"
	StatusMissing    SecretStatus = "missing"
)

type SecretItem struct {
	Key         string       `json:"key"`
	Source      string       `json:"source"`
	Status      SecretStatus `json:"status"`
	Description string       `json:"description,omitempty"`
}

type SecretAuditReport struct {
	WorkspaceRoot string       `json:"workspace_root"`
	Provider      string       `json:"provider,omitempty"`
	Secrets       []SecretItem `json:"secrets"`
	MissingCount  int          `json:"missing_count"`
	FoundCount    int          `json:"found_count"`
}

// standardSystemEnvVars lists common runtime/system environment variables
// that should not be flagged as missing application secrets.
var standardSystemEnvVars = map[string]bool{
	"NODE_ENV":              true,
	"PORT":                  true,
	"HOST":                  true,
	"HOSTNAME":              true,
	"CI":                    true,
	"VERCEL":                true,
	"VERCEL_ENV":            true,
	"NETLIFY":               true,
	"RENDER":                true,
	"PATH":                  true,
	"PWD":                   true,
	"HOME":                  true,
	"USER":                  true,
	"SHELL":                 true,
	"TERM":                  true,
	"LANG":                  true,
	"LC_ALL":                true,
	"TMPDIR":                true,
	"TEMP":                  true,
	"TMP":                   true,
	"SHLVL":                 true,
	"_":                     true,
	"EDITOR":                true,
	"VISUAL":                true,
	"PAGER":                 true,
	"PS1":                   true,
	"LOGNAME":               true,
	"COLORTERM":             true,
	"TERM_PROGRAM":          true,
	"TERM_PROGRAM_VERSION":  true,
	"SYSTEMROOT":            true,
	"WINDIR":                true,
	"COMSPEC":               true,
	"PATHEXT":               true,
	"PROMPT":                true,
	"HOMEDRIVE":             true,
	"HOMEPATH":              true,
	"USERPROFILE":           true,
	"APPDATA":               true,
	"LOCALAPPDATA":          true,
	"PROGRAMDATA":           true,
	"PROGRAMFILES":          true,
	"PROGRAMFILES(X86)":     true,
	"COMMONPROGRAMFILES":    true,
	"COMMONPROGRAMFILES(X86)": true,
}

var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".next":        true,
	"dist":         true,
	"build":        true,
	".atlas":       true,
	"__pycache__":  true,
	".venv":        true,
	".turbo":       true,
	".output":      true,
}

var validKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Source code regex patterns for extracting environment variable references
var codePatterns = []*regexp.Regexp{
	// JS / TS
	regexp.MustCompile(`process\.env\.([A-Za-z0-9_]+)`),
	regexp.MustCompile(`process\.env\[['"]([A-Za-z0-9_]+)['"]\]`),
	// Python
	regexp.MustCompile(`os\.environ\.get\(['"]([A-Za-z0-9_]+)['"]\)`),
	regexp.MustCompile(`os\.environ\[['"]([A-Za-z0-9_]+)['"]\]`),
	regexp.MustCompile(`os\.getenv\(['"]([A-Za-z0-9_]+)['"]\)`),
	// Go
	regexp.MustCompile(`os\.Getenv\(['"]([A-Za-z0-9_]+)['"]\)`),
	regexp.MustCompile(`os\.LookupEnv\(['"]([A-Za-z0-9_]+)['"]\)`),
	// Ruby
	regexp.MustCompile(`ENV\[['"]([A-Za-z0-9_]+)['"]\]`),
	regexp.MustCompile(`ENV\.fetch\(['"]([A-Za-z0-9_]+)['"]\)`),
	// PHP
	regexp.MustCompile(`\$_ENV\[['"]([A-Za-z0-9_]+)['"]\]`),
	regexp.MustCompile(`getenv\(['"]([A-Za-z0-9_]+)['"]\)`),
}

var supportedExtensions = map[string]bool{
	".js":   true,
	".jsx":  true,
	".ts":   true,
	".tsx":  true,
	".mjs":  true,
	".cjs":  true,
	".py":   true,
	".go":   true,
	".rb":   true,
	".php":  true,
}

var templateFileNames = []string{
	".env.example",
	".env.sample",
	".env.template",
	".env.local.example",
}

// AuditWorkspace scans the given root directory (and parent monorepo root if detected)
// for environment variable requirements from template files and source code references,
// evaluating whether each is present locally or in provider configuration.
// No secret values are ever accessed, printed, or saved.
func AuditWorkspace(root string, provider string, providerEnvKeys []string) (*SecretAuditReport, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = filepath.Clean(root)
	}

	report := &SecretAuditReport{
		WorkspaceRoot: absRoot,
		Provider:      provider,
		Secrets:       make([]SecretItem, 0),
	}

	// Determine if part of a monorepo
	var monorepoRoot string
	if ws, err := workspace.Resolve(absRoot); err == nil && ws != nil {
		if ws.MonorepoRoot != "" && ws.MonorepoRoot != absRoot {
			monorepoRoot = ws.MonorepoRoot
		}
	}

	// key -> set of sources
	keySources := make(map[string][]string)

	// 1. Scan template files in workspace root
	for _, tpl := range templateFileNames {
		tplPath := filepath.Join(absRoot, tpl)
		if keys, err := parseTemplateKeys(tplPath); err == nil {
			for _, k := range keys {
				addKeySource(keySources, k, tpl)
			}
		}
	}

	// Also scan template files in monorepo root if applicable
	if monorepoRoot != "" {
		for _, tpl := range templateFileNames {
			tplPath := filepath.Join(monorepoRoot, tpl)
			if keys, err := parseTemplateKeys(tplPath); err == nil {
				for _, k := range keys {
					addKeySource(keySources, k, tpl+" (monorepo)")
				}
			}
		}
	}

	// 2. Scan source code in workspace root
	_ = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExtensions[ext] {
			return nil
		}

		relPath, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			relPath = d.Name()
		}
		relPath = filepath.ToSlash(relPath)

		keys, scanErr := scanSourceFile(path)
		if scanErr == nil {
			for _, k := range keys {
				addKeySource(keySources, k, relPath)
			}
		}
		return nil
	})

	// 3. Collect configured keys from .env files without reading values
	envKeys := make(map[string]bool)
	localEnvFiles := []string{
		".env",
		".env.local",
		".env.production",
		".env.production.local",
	}
	for _, f := range localEnvFiles {
		if kMap, err := parseEnvFileKeys(filepath.Join(absRoot, f)); err == nil {
			for k := range kMap {
				envKeys[k] = true
			}
		}
	}
	if monorepoRoot != "" {
		for _, f := range localEnvFiles {
			if kMap, err := parseEnvFileKeys(filepath.Join(monorepoRoot, f)); err == nil {
				for k := range kMap {
					envKeys[k] = true
				}
			}
		}
	}

	providerKeyMap := make(map[string]bool)
	for _, pk := range providerEnvKeys {
		providerKeyMap[pk] = true
	}

	// 4. Status Evaluation
	for key, sources := range keySources {
		if standardSystemEnvVars[key] {
			continue
		}

		status := StatusMissing
		if providerKeyMap[key] {
			status = StatusConfigured
		} else if envKeys[key] || os.Getenv(key) != "" {
			status = StatusFoundLocal
		}

		sourceSummary := strings.Join(sources, ", ")

		report.Secrets = append(report.Secrets, SecretItem{
			Key:    key,
			Source: sourceSummary,
			Status: status,
		})

		if status == StatusMissing {
			report.MissingCount++
		} else {
			report.FoundCount++
		}
	}

	// Sort secrets deterministically by Key
	sort.Slice(report.Secrets, func(i, j int) bool {
		return report.Secrets[i].Key < report.Secrets[j].Key
	})

	return report, nil
}

func addKeySource(m map[string][]string, key, source string) {
	if standardSystemEnvVars[key] {
		return
	}
	existing := m[key]
	for _, s := range existing {
		if s == source {
			return
		}
	}
	m[key] = append(existing, source)
}

// parseTemplateKeys extracts variable names from lines like `KEY=...` or `KEY=`.
// Comments (#) and blank lines are ignored.
func parseTemplateKeys(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keys []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		if validKeyRegex.MatchString(key) {
			keys = append(keys, key)
		}
	}
	return keys, scanner.Err()
}

// parseEnvFileKeys extracts variable names defined in a .env file.
// Values are completely ignored and not stored.
func parseEnvFileKeys(filePath string) (map[string]bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	keys := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			if validKeyRegex.MatchString(key) {
				keys[key] = true
			}
		}
	}
	return keys, scanner.Err()
}

// scanSourceFile reads source code and extracts environment variable names.
func scanSourceFile(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var keys []string

	for _, pattern := range codePatterns {
		matches := pattern.FindAllStringSubmatch(string(content), -1)
		for _, m := range matches {
			if len(m) > 1 {
				key := m[1]
				if validKeyRegex.MatchString(key) && !seen[key] {
					seen[key] = true
					keys = append(keys, key)
				}
			}
		}
	}

	return keys, nil
}
