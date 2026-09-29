package cmd

import (
	"fmt"
	"os"

	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/updater"
	"github.com/Yashh56/atlas/internal/version"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Atlas to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		cliutil.PrintWelcome()

		cliutil.Info("Checking for updates...")
		rel, err := updater.CheckUpdate(version.Version)
		if err != nil {
			cliutil.Error(fmt.Sprintf("Failed to check for updates: %v", err))
			os.Exit(1)
		}

		if rel == nil {
			cliutil.Success(fmt.Sprintf("Atlas is up to date (v%s)", version.Version))
			return
		}

		cliutil.Info(fmt.Sprintf("Downloading Atlas %s...", rel.TagName))

		err = updater.PerformUpdate(rel)
		if err != nil {
			cliutil.Error(fmt.Sprintf("Failed to install update: %v", err))
			os.Exit(1)
		}

		cliutil.Success(fmt.Sprintf("Atlas was successfully updated to %s", rel.TagName))
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
