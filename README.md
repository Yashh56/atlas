# Atlas

**Atlas** is an autonomous deployment agent — a CLI tool that analyzes your project, runs the build, uses an LLM to auto-fix build errors, and deploys to a cloud provider, all in one command.

---

## Getting Started

### 1. Install

```bash
git clone https://github.com/Yashh56/atlas
cd atlas
go build -o atlas ./cmd/atlas
```

### 2. Set your LLM API key

Atlas uses an LLM (by default Anthropic Claude) to auto-fix build errors. Set the key for your chosen provider:

**Using the CLI (Secure Credential Store):**
```bash
# Example: Store your Mistral key securely
atlas models set mistral
```

**Using Environment Variables:**
```bash
# Example: Mistral
export MISTRAL_API_KEY=your-key-here

# Or add to .env in your project root:
echo 'MISTRAL_API_KEY=your-key-here' >> .env
```

### 3. Verify everything is ready

**Run these first** — they tell you whether Atlas can actually work:

```bash
# Check your LLM API keys
atlas models

# Check your deploy provider credentials  
atlas providers
```

These are the real answers to "is this going to work?" before you run a deploy.

### 4. Configure your project

Create `.atlas/config.json` in your project root:

```json
{
  "llm_provider": "mistral",
  "default_model": "mistral-large-latest",
  "approval": "manual"
}
```

### 5. Deploy

```bash
# Run interactively (wizard will prompt for provider & action)
atlas ./my-project

# Or run fully non-interactive via flags
atlas ./my-project --action deploy --provider render
```

Atlas will:
1. Check Vercel auth (prompt to install/login if needed)
2. Detect your framework and build command
3. Run the build
4. If build fails, call the LLM to fix it and retry (up to 4 times)
5. Ask for approval, then deploy

---

## Commands

| Command | Description |
|---------|-------------|
| `atlas <path> --action deploy --provider <name>` | Full deploy pipeline |
| `atlas providers` | Check deploy provider auth status |
| `atlas models` | Check LLM API key status |
| `atlas testllm <path>` | Verify LLM key with a live ping |
| `atlas debug run-command <path> -- <cmd>` | Run a command in a workspace |

See [`docs/COMMANDS.md`](docs/COMMANDS.md) for full documentation with example output.

---

## Supported Providers

### Deploy
| Provider | Status |
|----------|--------|
| Vercel | ✓ Implemented |
| Render | Planned |
| Netlify | Planned |
| Fly.io | Planned |
| Railway | Planned |

### LLM
| Provider | Env var |
|----------|---------|
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| Groq | `GROQ_API_KEY` |
| xAI (Grok) | `XAI_API_KEY` |
| Local (Ollama) | No key needed |

---

## Architecture

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full package layout and design rationale, including why credentials are global but session state is project-local.
