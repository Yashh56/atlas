<div align="center">
  <img src="public/logo.svg" alt="Atlas Logo" width="200"/>
  <h1>Atlas</h1>
  <p><strong>The Autonomous Deployment Pipeline</strong></p>
  
  [![Go Version](https://img.shields.io/github/go-mod/go-version/Yashh56/atlas?style=flat-square&color=00ADD8)](https://go.dev/)
  [![Release](https://img.shields.io/github/v/release/Yashh56/atlas?style=flat-square&color=green)](https://github.com/Yashh56/atlas/releases)
  [![License](https://img.shields.io/github/license/Yashh56/atlas?style=flat-square)](LICENSE)

  <p>
    Atlas analyzes your project, runs the build, uses an LLM to auto-fix build errors, and deploys to a cloud provider—all in a single command. 
  </p>
  <br/>
</div>

---

## ✨ Features

- **🧠 Auto-Healing Builds:** When a build fails, Atlas diagnoses the error, suggests a patch using an LLM (Anthropic, Mistral, Groq, etc.), and retries autonomously.
- **🚀 One-Command Deploy:** From local directory to live URL without opening the browser.
- **🔌 Multi-Provider Support:** First-class deployments for Vercel, Render, and Netlify.
- **🔎 Framework Detection:** Automatically detects React, Next.js, Express, and Vite projects and runs the correct build tools.
- **🛡️ Secure Credential Management:** Employs the OS keychain (via `go-keyring`) to store your sensitive LLM and Provider API keys safely.
- **🤖 Deterministic Pipeline:** Not a wild looping agent. Atlas follows a strictly bounded, predictable state machine for safe deployments.

---

## ⚡ Getting Started

### 1. Installation

You can install Atlas using our seamless install script (Linux/macOS):

```bash
curl -sSL https://raw.githubusercontent.com/Yashh56/atlas/master/install.sh | sh
```

*Or via Go directly:*
```bash
go install github.com/Yashh56/atlas/cmd/atlas@latest
```

### 2. Set your LLM API key

Atlas requires an LLM to fix broken builds. We support Anthropic, OpenAI, Mistral, Gemini, Groq, and xAI.

```bash
# Example: Store your Anthropic key securely via the CLI wizard
atlas models set anthropic
```
*You can also use environment variables like `export ANTHROPIC_API_KEY=...`*

### 3. Deploy Your Project

Navigate to any supported project directory and let Atlas do the magic:

```bash
cd ./my-nextjs-app
atlas .
```

Atlas will interactively ask you to choose a provider (e.g., Vercel), authenticate if you aren't already, run the build, fix any errors, and give you a live URL.

*(For CI/CD usage, you can run non-interactively: `atlas . --action deploy --provider vercel`)*

---

## 📚 Commands

| Command                                          | Description                       |
| ------------------------------------------------ | --------------------------------- |
| `atlas <path>`                                   | Interactive deployment wizard     |
| `atlas <path> --action deploy --provider <name>` | Full non-interactive pipeline     |
| `atlas providers`                                | Check deploy provider auth status |
| `atlas models`                                   | Check LLM API key status          |
| `atlas testllm <path>`                           | Verify LLM key with a live ping   |

*For a full breakdown of the architecture, see our [Documentation](docs/content/docs/architecture.mdx).*

---

## 🌐 Supported Integrations

### Frameworks

| Framework | Status |
| --------- | ------ |
| NextJS    | ✅     |
| React     | ✅     |
| Vite      | ✅     |
| Express   | ✅     |
| Django    | ⏳ Planned |

### Deployment Providers

| Provider | Status |
| -------- | ------ |
| Vercel   | ✅     |
| Render   | ✅     |
| Netlify  | ✅     |
| Fly.io   | ⏳ Planned |
| Railway  | ⏳ Planned |

---

## ❓ FAQ

**Q: Does Atlas rewrite my code unexpectedly?**  
A: No. Atlas uses exact-match patching. It does not rewrite entire files and will explicitly ask for approval before continuing the deployment after a patch.

**Q: Where are my API keys stored?**  
A: Atlas stores credentials securely in your operating system's native keychain (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux) and never writes them to plaintext files.

**Q: Can I use local models?**  
A: Yes! You can configure Atlas to use local LLMs (like Ollama) for completely free, private auto-healing.

---

## 🗺️ Future Plans

- **Docker Support:** Automatically generate `Dockerfile`s for unknown frameworks.
- **Rollback Mechanics:** Automatically revert to a previous healthy deployment if the post-deployment health check fails.
- **More Frameworks:** Deep integrations for Django, Spring Boot, and Laravel.
- **Multi-Environment Deployments:** Native support for staging and production branching.

---

<div align="center">
  <sub>Built with ❤️ by the Atlas Team.</sub>
</div>
