# Atlas Architecture

Atlas is an autonomous deployment agent — a Go CLI that takes a project directory, runs the analysis → build → fix → deploy pipeline, and manages credentials and LLM calls along the way.

---

## Package Layout

```
atlas/
├── cmd/
│   └── atlas/
│       ├── main.go          Entry point — loads .env, calls cmd.Execute()
│       └── cmd/             Cobra command wiring
│           ├── root.go      Root command + subcommand registration
│           ├── deploy.go    `atlas deploy <path> --provider <name>`
│           ├── debug.go     `atlas debug run-command`
│           ├── testllm.go   `atlas testllm <path>` — verify LLM key
│           ├── providers.go `atlas providers` — deploy auth status
│           └── models.go    `atlas models` — LLM key status
│
├── internal/
│   ├── config/        Project-local configuration (.atlas/config.json)
│   ├── credentials/   Global per-user credential store (OS keychain)
│   ├── workspace/     Workspace resolution (path validation, git root)
│   ├── session/       Session lifecycle and ID management
│   ├── state/         JSON persistence helpers (project.json, build.json, …)
│   ├── build/         Build command detection and execution
│   ├── detector/      Framework/package manager detection
│   ├── tools/         Individual pipeline tools (AnalyzeProject, RunBuildCommand, FixCode, …)
│   ├── llm/           GoAI-based LLM client and provider resolution
│   ├── deploy/        Deployment providers (Vercel) + EnsureVercelAuth
│   ├── orchestrator/  Pipeline orchestration (the main loop)
│   ├── registry/      Provider registry
│   ├── healthcheck/   HTTP health check helpers
│   └── audit/         Audit logging
│
├── skills/
│   └── fix_build.md   System prompt for the fix-code LLM call
│
└── docs/
    ├── ARCHITECTURE.md  (this file)
    └── COMMANDS.md      CLI reference with real example output
```

---

## Internal Subsystems

### `internal/config` — Project-local configuration
Owns `.atlas/config.json` per project. Defines LLM provider, model, local LLM base URL, and approval mode. If the file doesn't exist, sane defaults are returned. Config is intentionally **project-local**, not global, because different projects may use different LLM providers or models.

### `internal/credentials` — Global per-user credential store
**Deliberately global** (one store per OS user, at `%AppData%\atlas` on Windows), because deploy credentials (Vercel tokens, API keys) are per-user, not per-project. Storing them per-project would mean re-authenticating for every new repo.

The metadata file (`credentials.json`) tracks *which* method is being used and *when* it was verified — but **never stores a raw secret**. Secrets live exclusively in the OS keychain via [go-keyring](https://github.com/zalando/go-keyring), with an automatic fallback to a `0600` file when no keychain backend is available (e.g., headless Linux servers).

This is the **only** Atlas subsystem that is global rather than project-local.

### `internal/workspace` — Workspace resolution
Validates that a given path exists and identifies the git root (by walking upward looking for `.git`). The resolved `Workspace` struct is the input boundary for the orchestrator — nothing downstream deals with raw paths.

### `internal/session` — Session lifecycle
Creates a timestamped session directory under `.atlas/sessions/<id>/` for each `atlas deploy` run. Each session gets its own log directory, context JSON files, and planner state. Sessions are **project-local** — they're scoped to the workspace they were started against.

### `internal/state` — JSON persistence
Thin helpers (`LoadJSON`, `SaveJSON`) for reading and writing session context files (`project.json`, `build.json`, `planner.json`, `deployment.json`). These files are the pipeline's "shared memory" — each tool reads inputs written by previous steps.

**Context file ownership rule**: each context file is owned by exactly one tool:
- `project.json` → `AnalyzeProject`
- `build.json` → `RunBuildCommand`
- `planner.json` → `orchestrator` (planner, not a tool)
- `deployment.json` → `orchestrator` (deployment record, tracking `LastHealthyDeployment`)

No tool writes to a context file it doesn't own.

### `internal/tools` — Pipeline tools
Each tool implements the `Tool` interface (`Name() string`, `Execute(ctx, sess) (ToolResult, error)`). Tools are pure functions — they read inputs from session context files and write outputs back. They don't call each other directly.

Currently implemented tools:
- `AnalyzeProject` — framework/package manager detection
- `GitValidate` — checks for clean working tree, extracts commit SHA for rollback
- `RunBuildCommand` — executes the framework-specific build command, captures logs
- `FixCode` — calls the LLM (GoAI `GenerateObject`) with the build log and skill prompt, writes the patched file
- `WriteFile` — atomic file write with workspace-root sandboxing
- `RunCommand` — generic shell command execution

### `internal/llm` — LLM provider resolution
Wraps [GoAI v0.8.6](https://github.com/zendev-sh/goai) to provide:
- `ResolveModel(cfg)` — maps `config.LLMProvider` to a `provider.LanguageModel`
- `GenerateStructured[T](ctx, model, system, user)` — typed structured generation (used by `FixCode`)
- `NewClient(cfg)` → `Client.Complete(ctx, system, user)` — plain text generation (used by `testllm`)

Supported providers: `anthropic`, `openai`, `gemini`, `mistral`, `groq`, `grok`, `local` (OpenAI-compatible).

### `internal/deploy` — Deployment providers
Implements `Provider` interface (`Name()`, `Deploy(...)`, `HealthCheck(...)`, `Rollback(...)`). Supports providers like Vercel, Netlify, and Render.

Also houses `EnsureVercelAuth`, `EnsureRenderAuth`, and `EnsureNetlifyAuth`, the pre-flight auth checks wired into the orchestrator. It checks credentials in priority order (env var → stored token → CLI delegation) before any build work starts, so a misconfigured credential fails fast rather than after a multi-minute build.

### `internal/orchestrator` — Pipeline orchestration
The central loop. Stages:
1. Load config
2. Resolve workspace
3. Resolve LLM model + open credential store
4. **Pre-flight auth check** (`EnsureVercelAuth` for Vercel)
5. Create session
6. Analyze project
7. Validate git state
8. Build → on failure: fix (LLM) → rebuild (loop, max 4 retries)
9. Approval gate (manual or auto)
10. Deploy
11. Record deployment URL
12. **Post-deploy HealthCheck**: Verify application health (`HTTP 200`)
13. **Rollback**: If the health check fails, prompt to rollback to the `LastHealthyDeployment` (or automatically rollback if `--auto-rollback-on-unhealthy` is passed).

On build exhaustion, the orchestrator reverts to the original git commit SHA captured in step 7.

---

## Retry and Escalation Model

The build→fix loop has a configurable maximum retry count (default: 4). On each failure:
1. The build log tail (last 40 lines) is sent to the LLM with the `skills/fix_build.md` system prompt.
2. The LLM returns a `FixResponse` struct (file path + new content + reasoning).
3. The file is written and the build is re-run.

If the LLM call itself fails (e.g., no API key), `FixCode` returns a `ToolResult` with `Success: false` and a descriptive `Error` string — never a silent blank error.

If all retries are exhausted, the orchestrator reverts the workspace to the pre-deploy commit SHA via `git checkout <sha> -- .` and returns a non-zero exit code.

---

## Why credentials are global but session state is project-local

**Session state** tracks _what happened in this deploy run_ for _this project_: build logs, git SHA, detected framework, deployment URL. This is inherently project-specific — the build log for Project A is useless for Project B.

**Credentials** track _who the user is_ with _which provider_. A Vercel account is tied to the developer, not to any one project. Storing it per-project would mean running `vercel login` once per repo, which is the exact friction we're eliminating.

The credential store sits at the OS user config level (`%AppData%\atlas`, `~/.config/atlas`) for the same reason that your SSH keys sit in `~/.ssh`, not in each repo's `.git`.
