# Atlas CLI UI Architecture

The Atlas CLI implements a unified, centralized design system built on top of the Charm ecosystem (Bubble Tea, Lip Gloss, Bubbles, and Huh). This document explains the architecture and how to use it when building new commands or modifying existing ones.

## Principles

1. **One Source of Truth for Design**: All colors, typography, borders, and spacing must come from `internal/cliutil`. Do not use `lipgloss.NewStyle()` or `lipgloss.Color()` directly in command files.
2. **Semantic Meaning**: Use semantic styles (`StyleSuccess`, `StyleError`, `StyleMuted`) rather than raw colors (e.g., green or red).
3. **Decoupled Presentation**: Business logic (e.g., in `internal/orchestrator`) should emit structured events or use high-level logger methods (`cliutil.Success`, `cliutil.StepLog`) instead of printing raw strings.
4. **Environment Awareness**: The UI automatically degrades in non-interactive environments (CI, JSON output) via `cliutil.NonInteractive`.

## Core Components (`internal/cliutil`)

### 1. Theme (`theme.go`)
The central palette based on Atlas's branding (indigo/cyan/purple).
- **Colors**: `Primary`, `Secondary`, `Success`, `Error`, `Warning`, `Info`, `Muted`, `Text`, `Border`, `Background`

### 2. Styles (`styles.go`)
Pre-configured Lip Gloss styles mapped to the theme.
- **Typography**: `StyleTitle`, `StyleSubtitle`, `StyleBody`, `StyleMuted`, `StyleLabel`, `StyleValue`
- **Semantics**: `StyleSuccess`, `StyleError`, `StyleWarning`, `StyleInfo`
- **Specialty**: `StyleCode`, `StyleCommand`, `StyleURL`, `StyleHighlight`

### 3. Icons (`icons.go`)
Standardized symbols for consistency.
- `IconSuccess` (✓), `IconError` (✗), `IconWarning` (⚠), `IconInfo` (i), `IconArrow` (→), `IconDot` (•), `IconStep` (◆)

### 4. Output (`output.go` & `deploy_logger.go`)
Methods for standard terminal printing.
- **Simple Logs**: `Success(msg)`, `Error(msg)`, `Warning(msg)`, `Info(msg)`
- **Deploy Logger**: Used by the orchestrator to emit structured steps (e.g. `logger.StepLog()`, `logger.Section()`).

### 5. UI Elements
- **Spinners (`spinner.go`)**: Built on `bubbles/spinner`. Managed via `StartSpinner(msg)` and `spinner.Stop()`.
- **Boxes (`box.go`)**: Used for drawing panels or bordered content (e.g. `cliutil.Panel(title, text)`).
- **Tables (`table.go`)**: Simple key-value alignment (e.g. `cliutil.Table(rows)`).
- **Prompts (`prompt.go`)**: Interactive inputs built on `huh` (e.g., `PromptConfirm`, `PromptSecret`).

## Usage Guidelines

**1. Printing Statuses**
```go
cliutil.Success("Deployment successful")
cliutil.Info("Fetching dependencies...")
```

**2. Requesting Input**
```go
yes, err := cliutil.PromptConfirm("Are you sure you want to deploy?")
token, err := cliutil.PromptSecret("Enter your API token")
```

**3. Long Running Tasks**
```go
spinner := cliutil.StartSpinner("Building project...")
err := build.Execute()
spinner.Stop()
if err != nil {
    cliutil.Error("Build failed")
}
```

**4. Data Presentation**
```go
fmt.Println(cliutil.Panel("Deployment Info", cliutil.Table([][]string{
    {"Framework", "Next.js"},
    {"Duration", "12.5s"},
})))
```
