package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/audit"
	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/workspace"
)

var (
	auditProviderFlag  string
	auditStrictFlag    bool
	auditDirFlag       string
	auditWorkspaceFlag string
)

var auditCmd = &cobra.Command{
	Use:   "audit [path]",
	Short: "Audit workspace for required environment variables and secrets",
	Long:  `audit scans template files and source code for environment variable references and checks if they are configured without exposing or storing secrets.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAudit,
}

func init() {
	auditCmd.Flags().StringVar(&auditProviderFlag, "provider", "", "target deployment provider (remote key sync planned for v0.3.0)")
	auditCmd.Flags().BoolVar(&auditStrictFlag, "strict", false, "exit with code 1 if missing secrets found")
	auditCmd.Flags().StringVar(&auditDirFlag, "dir", "", "target subdirectory within a monorepo (e.g. apps/web)")
	auditCmd.Flags().StringVar(&auditWorkspaceFlag, "workspace", "", "target package name within a monorepo (e.g. web or @myorg/web)")
}

func runAudit(cmd *cobra.Command, args []string) error {
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	} else if pathFlag != "" {
		targetPath = pathFlag
	}

	dir := auditDirFlag
	if dir == "" {
		dir = dirFlag
	}
	wsPkg := auditWorkspaceFlag
	if wsPkg == "" {
		wsPkg = workspaceFlag
	}

	ws, err := workspace.ResolveWithSubdir(targetPath, dir, wsPkg)
	if err != nil {
		return err
	}
	if !ws.Exists {
		return fmt.Errorf("workspace path %q does not exist", targetPath)
	}

	report, err := audit.AuditWorkspace(ws.Root, auditProviderFlag, nil)
	if err != nil {
		return fmt.Errorf("secret audit: %w", err)
	}

	if jsonFlag {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		if auditStrictFlag && report.MissingCount > 0 {
			return fmt.Errorf("secret audit failed: %d missing secrets", report.MissingCount)
		}
		return nil
	}

	// Human-readable table output
	fmt.Println(cliutil.FormatHeader("PRE-FLIGHT SECRET AUDIT"))
	fmt.Printf("  %-12s %s\n", cliutil.StyleSubtext.Render("Target:"), cliutil.StyleHighlight.Render(ws.Root))
	if auditProviderFlag != "" {
		fmt.Printf("  %-12s %s (remote key sync planned for v0.3.0; evaluating local environment)\n", cliutil.StyleSubtext.Render("Provider:"), cliutil.StyleHighlight.Render(auditProviderFlag))
	}
	fmt.Println()

	if len(report.Secrets) == 0 {
		fmt.Printf("  %s No environment variables or secrets detected in templates or source code.\n\n", cliutil.IconInfo)
		return nil
	}

	maxKey := len("KEY")
	maxSource := len("SOURCE")
	for _, s := range report.Secrets {
		if len(s.Key) > maxKey {
			maxKey = len(s.Key)
		}
		if len(s.Source) > maxSource {
			maxSource = len(s.Source)
		}
	}

	fmt.Printf("  %-*s   %-*s   %s\n", maxKey, cliutil.StyleLabel.Render("KEY"), maxSource, cliutil.StyleLabel.Render("SOURCE"), cliutil.StyleLabel.Render("STATUS"))
	fmt.Printf("  %s   %s   %s\n", strings.Repeat("─", maxKey), strings.Repeat("─", maxSource), strings.Repeat("─", 14))

	for _, s := range report.Secrets {
		var statusStr string
		switch s.Status {
		case audit.StatusFoundLocal:
			statusStr = fmt.Sprintf("%s %s", cliutil.IconSuccess, cliutil.StyleSuccess.Render("found local"))
		case audit.StatusConfigured:
			statusStr = fmt.Sprintf("%s %s", cliutil.IconSuccess, cliutil.StyleSuccess.Render("configured"))
		case audit.StatusMissing:
			statusStr = fmt.Sprintf("%s %s", cliutil.IconError, cliutil.StyleError.Render("missing"))
		default:
			statusStr = string(s.Status)
		}

		fmt.Printf("  %-*s   %-*s   %s\n", maxKey, cliutil.StyleHighlight.Render(s.Key), maxSource, cliutil.StyleMuted.Render(s.Source), statusStr)
	}

	fmt.Println()
	if report.MissingCount > 0 {
		fmt.Printf("%s Found %d configured, %d missing environment variable(s).\n", cliutil.IconWarning, report.FoundCount, report.MissingCount)
	} else {
		fmt.Printf("%s Found %d configured, 0 missing environment variable(s).\n", cliutil.IconSuccess, report.FoundCount)
	}

	if auditStrictFlag && report.MissingCount > 0 {
		return fmt.Errorf("secret audit failed: %d missing secrets", report.MissingCount)
	}

	return nil
}
