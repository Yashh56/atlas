package cliutil

var (
	IconSuccess = StyleSuccess.Bold(true).Render("✓")
	IconError   = StyleError.Bold(true).Render("✗")
	IconWarning = StyleWarning.Bold(true).Render("⚠")
	IconInfo    = StyleInfo.Bold(true).Render("i")
	IconArrow   = StyleInfo.Bold(true).Render("→")
	IconDot     = StyleMuted.Render("•")
	IconStep    = StylePrimary.Render("◆")
)

var StylePrimary = StyleTitle
