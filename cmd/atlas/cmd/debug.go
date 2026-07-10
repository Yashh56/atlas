package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

var debugRunDir string

// debugCmd is a hidden parent for developer/debug subcommands.
var debugCmd = &cobra.Command{
	Use:    "debug",
	Short:  "Hidden debug utilities (not for end users)",
	Hidden: true,
}

// debugRunCommandCmd implements: atlas debug run-command "<cmd>" --dir <dir>
var debugRunCommandCmd = &cobra.Command{
	Use:   "run-command <command...>",
	Short: "Execute a shell command via the RunCommand tool",
	Long: `run-command executes the given command string using the RunCommand tool and
prints the captured output. Useful for exercising the tool without the full pipeline.

Example:
  atlas debug run-command "git status" --dir ./oss/todo
  atlas debug run-command echo hello`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Accept either a single quoted string ("git status") or multiple tokens.
		var command string
		var cmdArgs []string

		if len(args) == 1 {
			// Single argument — split on spaces so users can write "git status".
			parts := strings.Fields(args[0])
			if len(parts) == 0 {
				return fmt.Errorf("command argument is empty")
			}
			command = parts[0]
			cmdArgs = parts[1:]
		} else {
			command = args[0]
			cmdArgs = args[1:]
		}

		rc := tools.RunCommand{
			Command: command,
			Args:    cmdArgs,
			Dir:     debugRunDir,
		}

		// Create a minimal stub session — the tool doesn't use it in Week 1.
		stub := session.New("debug", debugRunDir)

		result, err := rc.Execute(context.Background(), stub)
		if err != nil {
			return fmt.Errorf("tool error: %w", err)
		}

		fmt.Print(result.Output)
		if !result.Success {
			fmt.Printf("\n[Error] %s\n", result.Error)
			return fmt.Errorf("command failed")
		}
		fmt.Printf("\n[Duration] %s\n", result.Duration)
		return nil
	},
}

func init() {
	debugRunCommandCmd.Flags().StringVar(&debugRunDir, "dir", ".", "working directory for the command")
	debugCmd.AddCommand(debugRunCommandCmd)
}
