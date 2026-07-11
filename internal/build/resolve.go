// Package build provides build-command resolution and execution for Atlas.
package build

import "fmt"

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
	case "nextjs", "react", "express":
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
	default:
		return "", nil, fmt.Errorf("don't know how to build framework: %q", framework)
	}
}
