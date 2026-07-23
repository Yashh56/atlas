// Package build provides build-command resolution and execution for Atlas.
package build

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveBuildCommand returns the command and arguments to build the project
// based on its detected framework and package manager. This is a pure function
// — no exec calls, no I/O, instantly testable.
//
// Supported combinations:
//
//	nextjs / react + pnpm    → pnpm build
//	nextjs / react + yarn    → yarn build
//	nextjs / react + npm/""  → npm run build
//	go                       → go build ./...
func ResolveBuildCommand(framework, packageManager string) (string, []string, error) {
	switch framework {
	case "nextjs", "react", "express", "node":
		switch packageManager {
		case "pnpm":
			return "pnpm", []string{"build"}, nil
		case "yarn":
			return "yarn", []string{"build"}, nil
		case "npm", "":
			return "npm", []string{"run", "build"}, nil
		default:
			return "npm", []string{"run", "build"}, nil
		}
	case "go":
		return "go", []string{"build", "./..."}, nil
	case "unknown", "":
		return "", nil, nil
	default:
		return "", nil, fmt.Errorf("don't know how to build framework: %q", framework)
	}
}

// ResolveTestCommand returns the command and arguments to run tests for the project
// based on its detected framework and package manager.
//
// Supported combinations:
//
//	nextjs / react + pnpm    → pnpm test
//	nextjs / react + yarn    → yarn test
//	nextjs / react + npm/""  → npm test
//	go                       → go test ./...
func ResolveTestCommand(framework, packageManager string) (string, []string, error) {
	switch framework {
	case "nextjs", "react", "express", "node":
		switch packageManager {
		case "pnpm":
			return "pnpm", []string{"test"}, nil
		case "yarn":
			return "yarn", []string{"test"}, nil
		case "npm", "":
			return "npm", []string{"test"}, nil
		default:
			return "npm", []string{"test"}, nil
		}
	case "go":
		return "go", []string{"test", "./..."}, nil
	case "unknown", "":
		return "", nil, nil
	default:
		return "", nil, fmt.Errorf("don't know how to test framework: %q", framework)
	}
}

// ResolvePublishDir heuristic-matches the build output directory for static hosting.
// Examples:
//   - react + vite.config.ts -> dist
//   - react + CRA -> build
//   - nextjs -> out (requires static export)
//   - go -> error (Netlify only serves static files)
func ResolvePublishDir(framework, packageManager, workspaceRoot string) (string, error) {
	if framework == "go" {
		return "", fmt.Errorf("Netlify only serves static output; a Go project isn't a valid target")
	}

	if framework == "nextjs" {
		// Next.js static exports go to "out" by default.
		return "out", nil
	}

	if framework == "react" || framework == "node" {
		// Heuristics for Vite
		hasViteConfig := false
		if _, err := os.Stat(filepath.Join(workspaceRoot, "vite.config.ts")); err == nil {
			hasViteConfig = true
		} else if _, err := os.Stat(filepath.Join(workspaceRoot, "vite.config.js")); err == nil {
			hasViteConfig = true
		}

		if hasViteConfig {
			return "dist", nil
		}
		
		// Fallback for Create React App or other standard builds
		if framework == "react" {
			return "build", nil
		}
	}

	return "", fmt.Errorf("could not determine the static output directory for framework %q. Please provide a manual --output-dir flag", framework)
}

