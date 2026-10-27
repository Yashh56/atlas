package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/doctor"
	"github.com/Yashh56/atlas/internal/workspace"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor [path]",
	Short: "Run diagnostics on environment tools, credentials, and LLM connectivity",
	Long:  "doctor inspects installed developer tools, OS keychain status, provider authentication, and LLM API connectivity.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var targetPath string
	if len(args) > 0 {
		targetPath = args[0]
	} else if pathFlag != "" {
		targetPath = pathFlag
	} else {
		targetPath = "."
	}

	ws, err := workspace.Resolve(targetPath)
	workspaceRoot := targetPath
	if err == nil && ws != nil {
		workspaceRoot = ws.Root
	}

	report, err := doctor.RunDiagnostics(cmd.Context(), workspaceRoot)
	if err != nil {
		return fmt.Errorf("doctor diagnostics failed: %w", err)
	}

	out := cmd.OutOrStdout()

	if jsonFlag {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(data))
		return nil
	}

	fmt.Fprintln(out, cliutil.FormatHeader("ATLAS DOCTOR"))

	for _, cat := range report.Categories {
		fmt.Fprintf(out, "\n%s\n", cliutil.StylePrompt.Render(cat.Category))
		for _, check := range cat.Checks {
			icon := cliutil.IconSuccess
			switch check.Status {
			case doctor.StatusWarn:
				icon = cliutil.IconWarning
			case doctor.StatusFail:
				icon = cliutil.IconError
			}

			info := check.Version
			if info == "" {
				info = check.Details
			} else if check.Details != "" {
				info = fmt.Sprintf("%s (%s)", check.Version, check.Details)
			}
			if check.Error != "" {
				info = fmt.Sprintf("%s — %s", info, cliutil.StyleError.Render(check.Error))
			}

			fmt.Fprintf(out, "  %s %-18s %s\n", icon, check.Name, cliutil.StyleSubtext.Render(info))
		}
	}

	fmt.Fprintln(out)
	if report.Overall == doctor.StatusPass {
		fmt.Fprintln(out, cliutil.FormatBox(fmt.Sprintf("%s All diagnostic checks passed! Atlas is ready.", cliutil.IconSuccess)))
	} else if report.Overall == doctor.StatusWarn {
		fmt.Fprintln(out, cliutil.FormatBox(fmt.Sprintf("%s Diagnostics completed with warnings. Atlas may still work for supported actions.", cliutil.IconWarning)))
	} else {
		fmt.Fprintln(out, cliutil.FormatBox(fmt.Sprintf("%s Critical diagnostic failures detected. Please fix the missing tools above.", cliutil.IconError)))
	}

	return nil
}
