// Package cmd wires together all Atlas CLI commands.
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "atlas",
	Short: "Atlas — autonomous deployment agent",
	Long:  "Atlas is a CLI tool that autonomously deploys your projects.",
}

// Execute is the entry point called from main.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(testllmCmd)
}
