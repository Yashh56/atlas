# AGENTS.md — Atlas

Read this before starting any task. It captures conventions established (often the hard way,
across real corrections) over the course of building Atlas — not aspirational style guidance, but
things that were tried differently once, broke or caused a review round, and got fixed.

## What Atlas actually is

A self-healing deployment pipeline, not a general-purpose coding agent. The vast majority of the
system is deterministic Go: framework detection, build/test/output-dir command resolution,
provider selection, retry policy, health-check evaluation, rollback decisions — all fixed logic,
zero LLM involvement. There is exactly **one** place a model makes a real judgment call:
`FixCode`, diagnosing a build failure and proposing a minimal patch, and even that step is tightly
bounded (a small tool-call step cap, exact-match patching, no free-form file rewrites).

**Default to deterministic code.** Before adding an LLM call anywhere, ask whether this is
genuinely a judgment call or just a lookup table / fixed policy in disguise. Nearly every time
this question has come up in this project, the answer was "fixed policy" — framework detection,
build commands, provider auth checks, rollback triggers were all initially candidates for "let
the model figure it out" and all turned out better as plain code. Don't assume this project wants
to become more agentic by default; it's gotten *more* deterministic as it matured, deliberately.

## Hard rules — not preferences, don't relitigate these per-task

1. **Verify external API/library behavior before coding against it.** `go doc`, a real docs fetch,
   or a live empirical test — never ship a guessed field name, endpoint, or response shape as
   settled fact. This project has shipped bugs from confidently-stated-but-unchecked claims
   multiple times (a provider's JSON field names, a rollback endpoint, a deploy-status enum, an
   API signature). If you find yourself writing "confirmed" or "verified" without having actually
   run the check, stop and run it.
2. **Generalize a pattern only after two real examples, not one.** Don't build a shared
   abstraction speculatively for hypothetical future cases — wait until a second concrete instance
   exists, then extract what the first one actually needed (established during the multi-LLM-
   provider work, and again for cross-provider auth resolution).
3. **Consequential or irreversible actions require explicit confirmation, always.** Pushing to a
   remote, installing something globally, connecting a git repo to a hosting provider, rolling
   back a live deployment — none of these ever happen silently, regardless of how confident the
   surrounding logic is. Defaults for anything in this category are off, not on.
4. **Never store secrets in plaintext JSON, logs, or any context file.** Credentials live in the
   OS keychain (`credentials.Store`) or environment variables only. If a value could be a token,
   key, or session secret, it does not appear in any `.atlas/sessions/**` file or log line.
5. **Autonomous code edits are exact-match patches, never full-file rewrites.** `PatchFile`'s
   `old_str`/`new_str` discipline exists specifically because full-file regeneration by an LLM
   will, under real conditions, "fix" an error by quietly discarding unrelated working code. This
   was observed directly, not theorized.
6. **Every loop involving an LLM or a retry needs a hard, enforced ceiling.** No loop should rely
   on "the model will eventually stop." Separately: a retry counter should distinguish *stuck*
   (same failure recurring, escalate fast) from *progressing* (a different failure each time,
   worth continuing) — treating every retry as identical either wastes budget on real progress or
   wastes budget on a truly unfixable loop.
7. **One state file, one owner.** Each context file (`session.json`, `planner.json`,
   `project.json`, `build.json`, `test.json`, `deployment.json`) belongs to one subsystem. A
   feature that needs to write two files in one step is a sign that step is doing two jobs.
8. **Check for an existing shared implementation before writing a parallel one.** Path-traversal
   guards, auth-resolution priority order, per-framework command tables — these should have one
   implementation reused across call sites, not near-duplicates that can silently drift apart.

## Perishable facts — do not trust anything here about specific versions

Provider model catalogs and API surfaces change fast — Groq alone had three deprecation waves in
under a year during this project. **Never hardcode a specific model string, endpoint, or field
name into this document as a fact to rely on** — the actual current values live in code
(`internal/llm/presets.go`, provider-specific clients) and should be treated as needing
reverification before trusting, no matter how recently they were checked. If a past task's plan
stated something was "verified as of [date]," treat that as a hint to recheck, not a guarantee.

## Workflow

- Non-trivial tasks get a written plan (`TASKS.md`-style) before implementation — architecture,
  proposed changes, open questions, verification plan. Flag genuine uncertainty explicitly rather
  than resolving it with a confident-sounding guess.
- Automated tests use fake transports / fake CLI interfaces — no real network calls in `go test
  ./...`. Anything touching a genuinely new external API surface needs at least one manual,
  empirical verification before being trusted, separate from the mocked test suite.
- Full architecture lives in `docs/ARCHITECTURE.md`, command reference in `docs/COMMANDS.md` —
  keep both current as part of any task that changes what they describe, not as a separate
  cleanup pass later.

## A note on runtime context (separate concern from this file)

This file is for whatever agent is developing Atlas. It has no bearing on what `FixCode` sees at
runtime when diagnosing a build in someone else's project — that's a separate, smaller idea
(should `FixCode` read a target project's own `CONTRIBUTING.md`/`AGENTS.md` if one exists, to
match that project's conventions?) worth considering later, not something this file addresses.
