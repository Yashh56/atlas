package build_test

import (
	"testing"

	"github.com/Yashh56/atlas/internal/build"
)

func TestResolveBuildCommand(t *testing.T) {
	cases := []struct {
		framework      string
		packageManager string
		wantCmd        string
		wantArgs       []string
		wantErr        bool
	}{
		// nextjs variants
		{"nextjs", "pnpm", "pnpm", []string{"build"}, false},
		{"nextjs", "yarn", "yarn", []string{"build"}, false},
		{"nextjs", "npm", "npm", []string{"run", "build"}, false},
		{"nextjs", "", "npm", []string{"run", "build"}, false},
		// react variants
		{"react", "pnpm", "pnpm", []string{"build"}, false},
		{"react", "yarn", "yarn", []string{"build"}, false},
		{"react", "npm", "npm", []string{"run", "build"}, false},
		// node generic
		{"node", "npm", "npm", []string{"run", "build"}, false},
		{"node", "pnpm", "pnpm", []string{"build"}, false},
		// go
		{"go", "", "go", []string{"build", "./..."}, false},
		{"go", "n/a", "go", []string{"build", "./..."}, false},
		// unknown
		{"rust", "", "", nil, true},
		{"", "", "", nil, false},
		{"unknown", "", "", nil, false},
		{"python", "pip", "", nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.framework+"/"+tc.packageManager, func(t *testing.T) {
			cmd, args, err := build.ResolveBuildCommand(tc.framework, tc.packageManager)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for framework=%q, got nil", tc.framework)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd != tc.wantCmd {
				t.Errorf("cmd = %q, want %q", cmd, tc.wantCmd)
			}
			if len(args) != len(tc.wantArgs) {
				t.Errorf("args = %v, want %v", args, tc.wantArgs)
				return
			}
			for i, a := range args {
				if a != tc.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, a, tc.wantArgs[i])
				}
			}
		})
	}
}

func TestResolveTestCommand(t *testing.T) {
	cases := []struct {
		framework      string
		packageManager string
		wantCmd        string
		wantArgs       []string
		wantErr        bool
	}{
		// nextjs variants
		{"nextjs", "pnpm", "pnpm", []string{"test"}, false},
		{"nextjs", "yarn", "yarn", []string{"test"}, false},
		{"nextjs", "npm", "npm", []string{"test"}, false},
		{"nextjs", "", "npm", []string{"test"}, false},
		// react variants
		{"react", "pnpm", "pnpm", []string{"test"}, false},
		{"react", "yarn", "yarn", []string{"test"}, false},
		{"react", "npm", "npm", []string{"test"}, false},
		// node generic
		{"node", "npm", "npm", []string{"test"}, false},
		{"node", "pnpm", "pnpm", []string{"test"}, false},
		// go
		{"go", "", "go", []string{"test", "./..."}, false},
		{"go", "n/a", "go", []string{"test", "./..."}, false},
		// unknown
		{"rust", "", "", nil, true},
		{"", "", "", nil, false},
		{"unknown", "", "", nil, false},
		{"python", "pip", "", nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.framework+"/"+tc.packageManager, func(t *testing.T) {
			cmd, args, err := build.ResolveTestCommand(tc.framework, tc.packageManager)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for framework=%q, got nil", tc.framework)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd != tc.wantCmd {
				t.Errorf("cmd = %q, want %q", cmd, tc.wantCmd)
			}
			if len(args) != len(tc.wantArgs) {
				t.Errorf("args = %v, want %v", args, tc.wantArgs)
				return
			}
			for i, a := range args {
				if a != tc.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, a, tc.wantArgs[i])
				}
			}
		})
	}
}
