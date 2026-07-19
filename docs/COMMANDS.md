# Atlas CLI Commands

All commands are invoked as `atlas <command> [flags]`.  
Run `atlas --help` for a full list at any time.

---

## `atlas`

Analyze, build, fix, run tests, and deploy a project.

**Usage:**
```
atlas [path] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--path <path>` | Project path (alternative to positional arg, use this for paths that collide with a subcommand name) |
| `--model <name>` | LLM provider to use (e.g., `anthropic`, `openai`) (optional) |
| `--action <mode>` | Action mode: `build`, `test`, `deploy`, `test-and-deploy` (optional) |
| `--provider <name>` | Deployment provider: `vercel`, `render`, `netlify`, `fly`, `railway` (optional) |
| `--allow-dirty` | Skip the uncommitted-changes check and proceed anyway |

*Note: Running `atlas <path>` interactively without all required flags will launch a terminal wizard to guide you through selecting a model, picking an action, and (conditionally) setting credentials and choosing a provider.*

**Path Collision Caveat:**
If your project directory shares a name with a registered subcommand (like `models`, `providers`, `debug`), `atlas <that-name>` will run the subcommand, not treat it as a path. Use the `--path <that-name>` escape hatch to disambiguate.

**Example — fully non-interactive, unchanged pipeline behavior:**
```
$ atlas ./oss/todo --model anthropic --action deploy --provider vercel
✓ Config loaded
✓ Workspace resolved
→ Checking Vercel authentication...
✓ Vercel authenticated (cli_session, you@example.com)
✓ Session created (sess_4881adb76fe6adb0)
✓ Project analyzed → framework: react, package_manager: npm
✓ Git validated → is_clean:true
✓ Build succeeded (12.3s) → dist/
→ Awaiting approval to deploy to vercel (production) [y/N]: y
→ Deploying to vercel (production)...
✓ Deployed successfully to https://my-app-xyz.vercel.app
```

**Example — interactive, nothing supplied:**
```
$ atlas ./oss/todo
# → model screen → action screen → (provider screen only if "deploy" or "test-and-deploy" chosen)
```

**Example — interactive, build-only:**
```
$ atlas ./oss/todo
# → model screen → action screen → user picks "Just build" → pipeline runs, NO provider screen
```

**Example — partial flags, wizard fills the rest:**
```
$ atlas ./oss/todo --model anthropic
# → wizard starts at the action screen, skips model select
```

**Example — build fails, LLM fixes it:**
```
✗ Build failed (attempt 1/4, 2.5s) → .atlas/sessions/sess_.../logs/build.log
→ Attempting automatic fix...
  Fixed: src/index.ts — "corrected missing semicolon on line 42"
→ Rebuilding...
✓ Build succeeded after 1 fix attempt(s) (3.1s)
```

---

## `atlas providers`

Show authentication status for all supported deploy providers.

**Usage:**
```
atlas providers
```

**Example output:**
```
$ atlas providers
DEPLOY PROVIDERS
  vercel     ✓ authenticated   (cli_session, you@example.com, verified 2026-07-13 10:02:00)
  render     ✗ not configured
```

**Subcommands:**
- `atlas providers set <provider>`: Securely store a deployment provider token.
- `atlas providers unset <provider>`: Remove a stored deployment provider token.

**Example — set a provider:**
```
$ atlas providers set vercel
Vercel token: ████████████████████████████
✓ Stored. This will be used instead of the CLI-delegated login next time you deploy.
```

**Example — unset a provider:**
```
$ atlas providers unset vercel
✓ Removed stored key for "vercel".
```

**Notes:**
- Vercel: checks `VERCEL_TOKEN` env var first, then the credential store.
- Other providers: checks their respective env vars (`RENDER_TOKEN`, `NETLIFY_AUTH_TOKEN`, `FLY_API_TOKEN`, `RAILWAY_API_TOKEN`). Credential store support for these is planned.
- Secret values are never printed — only `detected`/`authenticated`/`not configured` status.

---

## `atlas models`

Show API key status for all supported LLM providers.

**Usage:**
```
atlas models
```

**Example output:**
```
$ atlas models
LLM PROVIDERS   (active: mistral — from config.json's llm_provider)
  anthropic  ✓ stored
  openai     ✗ OPENAI_API_KEY not set
  gemini     ✗ GEMINI_API_KEY not set
```

**Subcommands:**
- `atlas models set <provider>`: Securely store an LLM API key.
- `atlas models unset <provider>`: Remove a stored LLM API key.

**Example — set a model:**
```
$ atlas models set anthropic
API key for anthropic: ████████████████████████████
✓ Stored. Run `atlas models` to confirm.
```

**Example — unset a model:**
```
$ atlas models unset anthropic
✓ Removed stored key for "anthropic".
```

**Notes:**
- The `active:` label comes from `.atlas/config.json`'s `llm_provider` field in the current directory. If no config is found, the label is omitted.
- Only env var presence is checked — the key is not validated by making an API call. Use `atlas testllm` to verify the key actually works.

---

## `atlas testllm`

Send a small ping to the configured LLM to verify the API key is active.

**Usage:**
```
atlas testllm <path>
```

**Example — working key:**
```
$ atlas testllm .
Testing LLM connection to provider: mistral (model: mistral-large-latest)...
LLM Response: SUCCESS
✓ API key is active and connection is working!
```

**Example — missing key:**
```
$ atlas testllm .
Testing LLM connection to provider: anthropic (model: claude-sonnet-4-6)...
Error: API call failed: goai generate text (anthropic): resolving auth token: goai: no API key or token source configured
```

---

## `atlas debug run-command`

Run an arbitrary shell command in a workspace and show output. Useful for debugging build detection.

**Usage:**
```
atlas debug run-command <path> -- <command> [args...]
```

**Example:**
```
$ atlas debug run-command ./my-app -- npm run build
> my-app@1.0.0 build
> vite build
...
```

---

## Setting Up Credentials

### LLM providers (via env vars or `.env`)

Set the API key for your chosen provider before running:
```
# Anthropic
ANTHROPIC_API_KEY=sk-ant-...

# Mistral
MISTRAL_API_KEY=...

# OpenAI
OPENAI_API_KEY=sk-...

# Groq
GROQ_API_KEY=gsk_...

# Gemini
GEMINI_API_KEY=...

# xAI (Grok)
XAI_API_KEY=...
```

Atlas automatically loads `.env` from the project's root via `godotenv`.

### Vercel (deploy provider)

**Option A — env var (CI/CD):**
```
VERCEL_TOKEN=<your-vercel-token>
```

**Option B — CLI interactive setup:**
```bash
atlas providers set vercel
# Prompts for masked input
```

**Option C — CLI login (interactive callback):**
```bash
atlas <path> --provider vercel
# Atlas will prompt to install/run `vercel login` if needed
```

### Config file (`.atlas/config.json`)

```json
{
  "llm_provider": "mistral",
  "default_model": "mistral-large-latest",
  "approval": "manual"
}
```

Supported `llm_provider` values: `anthropic`, `openai`, `gemini`, `mistral`, `groq`, `grok`, `local`.

For local/Ollama:
```json
{
  "llm_provider": "local",
  "default_model": "llama3",
  "local_llm_base_url": "http://localhost:11434/v1"
}
```
