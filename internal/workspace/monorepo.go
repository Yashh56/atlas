package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type WorkspaceType string

const (
	WorkspaceSingle WorkspaceType = "single"
	WorkspacePnpm   WorkspaceType = "pnpm"
	WorkspaceTurbo  WorkspaceType = "turborepo"
	WorkspaceNx     WorkspaceType = "nx"
	WorkspaceLerna  WorkspaceType = "lerna"
	WorkspaceNpm    WorkspaceType = "npm"
	WorkspaceYarn   WorkspaceType = "yarn"
	WorkspaceBun    WorkspaceType = "bun"
)

type PackageInfo struct {
	Name        string `json:"name"`
	RelPath     string `json:"rel_path"`     // e.g. "apps/web"
	AbsPath     string `json:"abs_path"`
	PackageJSON string `json:"package_json"` // path to package.json
}

type MonorepoInfo struct {
	Type     WorkspaceType `json:"type"`
	Root     string        `json:"root"`
	Packages []PackageInfo `json:"packages"`
}

// DetectMonorepo walks up from dir towards gitRoot looking for monorepo markers.
// Returns nil if no monorepo structure is detected.
func DetectMonorepo(dir, gitRoot string) *MonorepoInfo {
	current, err := filepath.Abs(dir)
	if err != nil {
		current = filepath.Clean(dir)
	}

	cleanGitRoot := ""
	if gitRoot != "" {
		if absGit, err := filepath.Abs(gitRoot); err == nil {
			cleanGitRoot = absGit
		} else {
			cleanGitRoot = filepath.Clean(gitRoot)
		}
	}

	for {
		wsType := checkDirForMonorepo(current)
		if wsType != "" {
			return &MonorepoInfo{
				Type:     wsType,
				Root:     current,
				Packages: DiscoverPackages(current),
			}
		}

		if cleanGitRoot != "" && current == cleanGitRoot {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return nil
}

func checkDirForMonorepo(dir string) WorkspaceType {
	if fileExists(filepath.Join(dir, "pnpm-workspace.yaml")) {
		return WorkspacePnpm
	}
	if fileExists(filepath.Join(dir, "turbo.json")) {
		return WorkspaceTurbo
	}
	if fileExists(filepath.Join(dir, "nx.json")) {
		return WorkspaceNx
	}
	if fileExists(filepath.Join(dir, "lerna.json")) {
		return WorkspaceLerna
	}

	pkgJSONPath := filepath.Join(dir, "package.json")
	if fileExists(pkgJSONPath) {
		if data, err := os.ReadFile(pkgJSONPath); err == nil {
			var raw struct {
				Workspaces any `json:"workspaces"`
			}
			if err := json.Unmarshal(data, &raw); err == nil && raw.Workspaces != nil {
				if fileExists(filepath.Join(dir, "bun.lockb")) || fileExists(filepath.Join(dir, "bun.lock")) {
					return WorkspaceBun
				}
				if fileExists(filepath.Join(dir, "yarn.lock")) {
					return WorkspaceYarn
				}
				if fileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
					return WorkspacePnpm
				}
				return WorkspaceNpm
			}
		}
	}

	return ""
}

// DiscoverPackages discovers packages inside a monorepo root.
func DiscoverPackages(monorepoRoot string) []PackageInfo {
	absRoot, err := filepath.Abs(monorepoRoot)
	if err != nil {
		absRoot = filepath.Clean(monorepoRoot)
	}

	var patterns []string
	patterns = append(patterns, parsePnpmWorkspace(absRoot)...)
	patterns = append(patterns, parsePackageJSONWorkspaces(absRoot)...)

	// Standard convention folders
	standardPatterns := []string{
		"apps/*",
		"packages/*",
		"services/*",
		"frontend",
		"backend",
	}
	patterns = append(patterns, standardPatterns...)

	var foundDirs []string
	for _, pat := range patterns {
		dirs := matchWorkspacePattern(absRoot, pat)
		foundDirs = append(foundDirs, dirs...)
	}

	seen := make(map[string]bool)
	var packages []PackageInfo

	for _, dir := range foundDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			absDir = dir
		}
		if seen[absDir] {
			continue
		}
		seen[absDir] = true

		pkgJSONPath := filepath.Join(absDir, "package.json")
		if !fileExists(pkgJSONPath) {
			continue
		}

		rel, err := filepath.Rel(absRoot, absDir)
		if err != nil {
			continue
		}
		relPath := filepath.ToSlash(rel)

		pkgName := ""
		if data, err := os.ReadFile(pkgJSONPath); err == nil {
			var p struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(data, &p); err == nil {
				pkgName = strings.TrimSpace(p.Name)
			}
		}
		if pkgName == "" {
			pkgName = filepath.Base(absDir)
		}

		packages = append(packages, PackageInfo{
			Name:        pkgName,
			RelPath:     relPath,
			AbsPath:     absDir,
			PackageJSON: pkgJSONPath,
		})
	}

	sort.Slice(packages, func(i, j int) bool {
		if packages[i].RelPath == packages[j].RelPath {
			return packages[i].Name < packages[j].Name
		}
		return packages[i].RelPath < packages[j].RelPath
	})

	if packages == nil {
		packages = []PackageInfo{}
	}
	return packages
}

func matchWorkspacePattern(root, pattern string) []string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || strings.HasPrefix(pattern, "!") {
		return nil
	}

	cleanPat := filepath.Clean(filepath.FromSlash(pattern))
	fullPattern := filepath.Join(root, cleanPat)

	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil
	}

	var res []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err == nil && info.IsDir() {
			res = append(res, m)
		}
	}
	return res
}

func parsePnpmWorkspace(root string) []string {
	content, err := os.ReadFile(filepath.Join(root, "pnpm-workspace.yaml"))
	if err != nil {
		return nil
	}
	var patterns []string
	inPackages := false
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "packages:") {
			inPackages = true
			continue
		}
		if inPackages {
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(trimmed, "-") {
				break
			}
			if strings.HasPrefix(trimmed, "-") {
				entry := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
				entry = strings.Trim(entry, `"'`)
				if entry != "" {
					patterns = append(patterns, entry)
				}
			}
		}
	}
	return patterns
}

func parsePackageJSONWorkspaces(root string) []string {
	content, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil
	}
	var raw struct {
		Workspaces any `json:"workspaces"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil
	}
	var patterns []string
	switch v := raw.Workspaces.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				patterns = append(patterns, strings.Trim(s, `"'`))
			}
		}
	case map[string]any:
		if pkgs, ok := v["packages"].([]any); ok {
			for _, item := range pkgs {
				if s, ok := item.(string); ok && s != "" {
					patterns = append(patterns, strings.Trim(s, `"'`))
				}
			}
		}
	}
	return patterns
}

// ResolveWorkspacePackage matches a name or relative path to a package in the monorepo.
func ResolveWorkspacePackage(monorepoRoot string, nameOrPath string) (*PackageInfo, error) {
	pkgs := DiscoverPackages(monorepoRoot)
	normalized := filepath.ToSlash(filepath.Clean(nameOrPath))

	// 1. Exact match on Name
	for _, p := range pkgs {
		if p.Name == nameOrPath {
			return &p, nil
		}
	}

	// 2. Exact match on RelPath
	for _, p := range pkgs {
		if p.RelPath == normalized {
			return &p, nil
		}
	}

	// 3. Case-insensitive match on Name
	for _, p := range pkgs {
		if strings.EqualFold(p.Name, nameOrPath) {
			return &p, nil
		}
	}

	// 4. Case-insensitive match on RelPath
	for _, p := range pkgs {
		if strings.EqualFold(p.RelPath, normalized) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("package %q not found in monorepo at %s", nameOrPath, monorepoRoot)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
