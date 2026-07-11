# Atlas — Week 3 build task

Week 2 gave you the four-file context model and a working `AnalyzeProject` step. This week: git
validation before build (populating the `git` fields reserved in `project.json` since Week 2),
real build execution with captured logs, and the build-success/failure branch — including proving
the retry counter in `planner.json` actually increments on failure, without yet building the LLM
fixer that would act on it.

Still **no LLM, no network calls except local `git`, no provider integration**. Everything this
week is deterministic Go and shelling out to local toolchains.

## 1. `GitValidate` tool

```go
type GitValidate struct {
    WorkspaceRoot string
}
func (g GitValidate) Name() string { return "git_validate" }
```

Populates the `git` block in `project.json` that Week 2 reserved but left null:

```json
"git": {
  "branch": "main",
  "commit_sha": "a1b2c3d...",
  "is_clean": true,
  "remote": "https://github.com/yash/todo.git"
}
```

Implementation:

- Reuse the Week 1 `RunCommand` tool internally rather than calling `os/exec` directly here —
  keep one code path for "run a shell command and capture output."
- `branch`: `git rev-parse --abbrev-ref HEAD`
- `commit_sha`: `git rev-parse HEAD`
- `is_clean`: `true` if `git status --porcelain` returns empty output, else `false`
- `remote`: `git remote get-url origin` — if this fails (no remote configured), leave `remote`
  null and don't treat it as an error. A missing remote is normal for a local-only repo.
- **If `Workspace.GitRoot` from Week 1 is empty** (not a git repo at all), don't fail — set all
  four fields to null, log a clear one-line warning, and let the pipeline continue. Plenty of
  early-stage projects aren't git-initialized yet and that's not this tool's problem to solve.

### Policy: block on a dirty tree by default

Add an `--allow-dirty` bool flag to `atlas deploy`. If `git.is_clean == false` and
`--allow-dirty` was not passed, the orchestrator aborts the pipeline **before** running any build
step, with a message like:

```
✗ Working tree has uncommitted changes. Commit or stash them, or re-run with --allow-dirty.
```

If the workspace isn't a git repo at all (`is_clean: null`), don't block — that's a different
situation from "dirty," and blocking on it would make Atlas unusable on non-git projects for no
good reason.

Acceptance: tests using a real temp git repo (`git init` via `RunCommand` inside the test setup,
not a mock) covering: clean repo → `is_clean: true`; a modified tracked file → `is_clean: false`;
no remote configured → `remote: null`, no error; a plain directory with no `.git` → all fields
null, no error. Also test the CLI-level abort behavior: dirty repo without the flag exits
non-zero with the message above; dirty repo with `--allow-dirty` proceeds.

## 2. Build command resolution — kept pure and separately testable

Split this into two pieces on purpose. Resolving *which command to run* should never require
actually running a process, which means it can be tested instantly without node/pnpm/go installed
in CI:

```go
// internal/build/resolve.go — pure function, no exec, no I/O
func ResolveBuildCommand(framework, packageManager string) (cmd string, args []string, err error)
```

Hardcoded table, matching Week 2's detection set:

| framework | package_manager | command |
|---|---|---|
| `nextjs` / `react` | `pnpm` | `pnpm build` |
| `nextjs` / `react` | `yarn` | `yarn build` |
| `nextjs` / `react` | `npm` or null | `npm run build` |
| `go` | — | `go build ./...` |

Anything else (unknown framework, or a framework that isn't in the table) returns a clear error —
`RunBuildCommand` treats that as an immediate pipeline failure with a message like
`"don't know how to build framework: <x>"`, not a panic or a silent no-op.

Acceptance: table-driven test over every row above plus an unknown-framework case, zero exec
calls, runs in milliseconds.

## 3. `RunBuildCommand` tool — the executor

```go
type RunBuildCommand struct {
    WorkspaceRoot  string
    Framework      string
    PackageManager string
}
func (r RunBuildCommand) Name() string { return "run_build_command" }
```

- Calls `ResolveBuildCommand` first; on error, return a failed `ToolResult` immediately without
  attempting to run anything.
- Executes via `context.WithTimeout` (5 minutes, hardcoded constant for now — a configurable
  timeout is a later concern) wrapping `RunCommand`.
- **Don't put the full build log in `ToolResult.Output` or in any JSON context file.** Write it to
  `.atlas/sessions/<id>/logs/build.log` and only reference the path. Build logs can be large;
  context files need to stay small and fast to rewrite atomically.
- Writes a new **`build.json`** context file (fifth file, owned by this tool):

```json
{
  "command": "pnpm build",
  "exit_code": 0,
  "duration_ms": 4213,
  "log_path": ".atlas/sessions/sess_a1b2c3d4/logs/build.log",
  "started_at": "2026-07-11T10:05:00Z"
}
```

Leave `output_dir` out of this schema for now — figuring out where the build artifact actually
landed is a Week 4 concern once you're wiring an actual provider that needs it.

Acceptance: this needs to be testable without requiring pnpm/yarn/npm actually installed in CI.
Test `RunBuildCommand` by injecting a resolved command directly (bypass `ResolveBuildCommand` in
the test, or test it as two layers) using something trivial and always-available like
`sh -c "exit 0"` for the success case and `sh -c "exit 1"` for the failure case. Assert:
`exit_code` and `duration_ms` land correctly in `build.json`, `build.log` is written and
non-empty, and the timeout path (use a 100ms context timeout against `sleep 5` in the test) marks
the result as failed rather than hanging.

## 4. Wire the build-success/failure branch into the orchestrator

Extend `internal/orchestrator` so `atlas deploy` now runs, in order: config → workspace → session
→ analyze project → git validate (with the dirty-tree abort check) → determine + run build.

**On build success**, update `planner.json`:
```json
{ "current_step": "build_complete", "completed": ["analyze_project", "git_validate", "run_build_command"] }
```
Print:
```
✓ Build succeeded (4.2s) → .atlas/sessions/sess_.../logs/build.log
Stopping here — deployment not implemented yet (Week 4).
```

**On build failure**, update `planner.json`:
```json
{
  "failed": ["run_build_command"],
  "retries": { "fix_and_rebuild": { "count": 1, "max": 4 } },
  "error": {
    "step": "run_build_command",
    "message": "<last ~20 lines of build.log, not the whole thing>",
    "occurred_at": "2026-07-11T10:05:04Z"
  }
}
```
Print:
```
✗ Build failed (attempt 1/4) → .atlas/sessions/sess_.../logs/build.log
No automatic fix available yet (Week 4). Exiting.
```
Exit non-zero. **Do not loop, do not retry the build automatically, do not attempt any fix.** The
counter incrementing to 1 and stopping is the entire scope here — it proves the state machine and
the schema are correct. The actual retry loop only makes sense once there's an LLM step that can
plausibly fix something between attempts; building the loop now just means looping over the same
guaranteed failure four times for no reason.

Acceptance: run against a real fixture project with a build script that always fails (a `go.mod`
project with a compile error, or a `package.json` with `"build": "exit 1"`) and confirm the
printed output, the exit code, and `planner.json`'s `retries.fix_and_rebuild.count == 1` all
match. Run against a fixture that succeeds and confirm the success path.

## Explicit non-goals for this week

- The LLM-backed Fix Code step, or any actual retry loop that runs the build more than once —
  Week 4+
- `output_dir` detection / build artifact packaging — Week 4, once a provider needs it
- Detect hosting, select deployment, any provider network call — Week 4
- Approval gate, health check — later
- Configurable build timeout (flag/config field) — hardcoded constant is fine for now
- A generic "any git provider" abstraction — you're shelling out to local `git`, that's it

If you find yourself about to build any of the above, stop and flag it instead of proceeding.

## Definition of done for Week 3

```bash
go build ./...
go test ./...   # zero network calls, zero dependency on node/pnpm/yarn being installed

# clean, buildable project:
go run ./cmd/atlas deploy ./fixtures/nextjs-ok --provider vercel
# → walks through analyze → git validate → build succeeds → stops, prints log path

# dirty working tree:
go run ./cmd/atlas deploy ./fixtures/dirty-repo --provider vercel
# → aborts before build with the git-dirty message; passes with --allow-dirty

# broken build:
go run ./cmd/atlas deploy ./fixtures/nextjs-broken --provider vercel
# → build fails, retries.fix_and_rebuild.count == 1 in planner.json, exits non-zero
```

Five context files now exist and round-trip correctly (`session.json`, `planner.json`,
`project.json`, `deployment.json`, `build.json`). Still no AI, no deployment — but the runtime can
now validate, build, and honestly report success or failure with a real audit trail on disk.