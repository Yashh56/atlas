import {
  Button,
  Accordion,
  Callout,
  Steps,
  Step,
  Tabs,
  Tab,
} from "@umami/shiso/components";

export const frontmatter = {
  title: "Atlas",
  description: "The autonomous, self-healing deployment pipeline.",
  search: false,
};

export default function Home() {
  return (
    <div className="flex flex-col items-center min-h-screen bg-gray-50 dark:bg-zinc-950 overflow-hidden relative">
      {/* Background ambient glow */}
      <div className="absolute top-0 inset-x-0 h-96 bg-gradient-to-b from-blue-500/20 to-transparent pointer-events-none blur-3xl opacity-50 dark:opacity-30"></div>

      {/* Hero Section */}
      <section className="w-full pt-32 pb-16 flex flex-col items-center text-center px-4 relative z-10">
        <h1 className="text-6xl md:text-8xl font-extrabold tracking-tighter mb-6">
          Meet{" "}
          <span className="bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-teal-400 drop-shadow-sm">
            Atlas
          </span>
        </h1>

        <p className="text-xl md:text-2xl text-gray-600 dark:text-gray-400 max-w-3xl mb-10 leading-relaxed font-medium">
          The autonomous deployment CLI that analyzes your project, executes
          local builds, diagnoses failures with LLMs, and safely ships to the
          cloud.
        </p>

        {/* Terminal example */}
        <div className="w-full max-w-xl mb-10 rounded-xl overflow-hidden shadow-2xl border border-gray-200 dark:border-zinc-700">
          <div className="bg-gray-800 dark:bg-zinc-800 px-4 py-2 flex items-center gap-2">
            <span className="w-3 h-3 rounded-full bg-red-500"></span>
            <span className="w-3 h-3 rounded-full bg-yellow-500"></span>
            <span className="w-3 h-3 rounded-full bg-green-500"></span>
            <span className="ml-2 text-sm text-gray-400 font-mono">
              terminal
            </span>
          </div>
          <pre className="bg-gray-900 dark:bg-zinc-900 text-green-400 p-6 text-left font-mono text-sm leading-relaxed overflow-x-auto">
            <code>{`$ atlas ./my-app

✓ Detected framework: Next.js
✓ Build command: npm run build
✓ Build succeeded
✓ Deployed to Vercel
✓ Health check passed

🚀 Live at: https://my-app.vercel.app`}</code>
          </pre>
        </div>

        <div className="flex flex-col sm:flex-row gap-4 w-full sm:w-auto mb-16">
          <Button
            href="/docs"
            icon="rocket"
            className="shadow-lg shadow-blue-500/25 hover:shadow-blue-500/40 hover:-translate-y-0.5 transition-all duration-300"
          >
            Get Started
          </Button>
          <Button
            href="https://github.com/Yashh56/atlas"
            variant="secondary"
            icon="github"
            className="hover:-translate-y-0.5 transition-all duration-300 bg-white/50 dark:bg-zinc-900/50 backdrop-blur-md"
          >
            View GitHub
          </Button>
        </div>
      </section>

      {/* Safety Callout */}
      <section className="w-full max-w-4xl px-4 py-8 relative z-10">
        <Callout variant="info" title="Safe by design">
          <p>
            Atlas never performs full-file rewrites. Every LLM-proposed fix is a
            sandboxed, exact-match patch — if it can't match precisely, it fails
            safely rather than guessing. All edits are backed up as file-level
            snapshots and automatically restored if the retry budget is
            exhausted. Consequential actions (pushing to remote, triggering a
            deploy) always require explicit confirmation.
          </p>
        </Callout>
      </section>

      {/* Pipeline Steps */}
      <section className="w-full max-w-4xl px-4 py-20 relative z-10">
        <div className="text-center mb-12">
          <h2 className="text-4xl font-bold tracking-tight mb-4">
            How Atlas Works
          </h2>
          <p className="text-gray-500 dark:text-gray-400 text-lg max-w-2xl mx-auto">
            A deterministic pipeline with one bounded AI step — not a
            free-roaming agent.
          </p>
        </div>

        <Steps>
          <Step title="Analyze">
            <p>
              Atlas reads your project structure, detects your framework
              (Next.js, React, Django, Express, Go), identifies your package
              manager, and resolves the correct build and start commands — zero
              config required.
            </p>
          </Step>
          <Step title="Build Locally">
            <p>
              Runs your build command in a sandboxed workspace. If it succeeds,
              Atlas proceeds directly to deployment.
            </p>
          </Step>
          <Step title="Fix (if needed)">
            <p>
              If the build fails, Atlas captures the full error output and sends
              it to your configured LLM (Claude, OpenAI, Gemini, Mistral, Groq,
              or a local Ollama model). The model proposes a fix as an
              exact-match string replacement — no full-file rewrites. Retries
              are strictly bounded (up to 4 attempts), with file-level snapshots
              restored automatically if the budget is exhausted.
            </p>
          </Step>
          <Step title="Deploy">
            <p>
              After a successful build, Atlas deploys to your chosen provider
              (Vercel, Render, or Netlify). For Django on Render, it
              auto-provisions a free-tier PostgreSQL database and wires the
              connection string.
            </p>
          </Step>
          <Step title="Health Check & Rollback">
            <p>
              Atlas runs an HTTP health check against your live URL with
              exponential-backoff retries. If the check fails and a prior
              healthy deployment exists, you are prompted to rollback.
            </p>
          </Step>
        </Steps>
      </section>

      {/* Non-sequential Capabilities */}
      <section className="w-full max-w-7xl px-4 py-16 relative z-10">
        <div className="text-center mb-12">
          <h2 className="text-4xl font-bold tracking-tight mb-4">
            Built-in Capabilities
          </h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {/* Database Provisioning */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-orange-100 dark:bg-orange-900/50 flex items-center justify-center text-orange-600 dark:text-orange-400 mb-6 group-hover:scale-110 transition-transform">
              <svg
                className="w-6 h-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                />
              </svg>
            </div>
            <h3 className="text-xl font-bold mb-3">Database Provisioning</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              Automatically provisions free-tier Postgres instances for Django
              on Render, and re-provisions them if manually deleted.
            </p>
          </div>

          {/* Secure Secrets */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-pink-100 dark:bg-pink-900/50 flex items-center justify-center text-pink-600 dark:text-pink-400 mb-6 group-hover:scale-110 transition-transform">
              <svg
                className="w-6 h-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                />
              </svg>
            </div>
            <h3 className="text-xl font-bold mb-3">Secure Secrets</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              Provider tokens and ephemeral keys (like Django's SECRET_KEY) are
              stored securely in your OS keychain, never saved to plaintext logs
              or files.
            </p>
          </div>

          {/* Multi-LLM Support */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-purple-100 dark:bg-purple-900/50 flex items-center justify-center text-purple-600 dark:text-purple-400 mb-6 group-hover:scale-110 transition-transform">
              <svg
                className="w-6 h-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                />
              </svg>
            </div>
            <h3 className="text-xl font-bold mb-3">Multi-LLM Support</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              Choose from Anthropic, OpenAI, Gemini, Mistral, Groq, xAI, or run
              completely offline with Ollama. Models are only used for the
              bounded FixCode diagnostic step.
            </p>
          </div>
        </div>
      </section>

      {/* Tech Stack Matrix */}
      <section className="w-full bg-white dark:bg-zinc-900 border-y border-gray-200 dark:border-zinc-800 py-16 px-4">
        <div className="max-w-5xl mx-auto flex flex-col md:flex-row gap-12 justify-around items-start">
          <div className="flex-1">
            <h3 className="text-sm font-bold tracking-widest uppercase text-gray-400 mb-6">
              Supported Frameworks
            </h3>
            <ul className="space-y-4">
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600 text-xs">
                  ✓
                </span>
                Next.js / React
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600 text-xs">
                  ✓
                </span>
                Express / Node.js
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600 text-xs">
                  ✓
                </span>
                Django{" "}
                <span className="text-sm text-gray-400 font-normal">
                  (Render and Vercel only)
                </span>
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600 text-xs">
                  ✓
                </span>
                Go
              </li>
            </ul>
          </div>

          <div className="flex-1">
            <h3 className="text-sm font-bold tracking-widest uppercase text-gray-400 mb-6">
              Supported Providers
            </h3>
            <ul className="space-y-4">
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-blue-100 text-blue-600 text-xs">
                  ✓
                </span>
                Render (Web Services & PostgreSQL)
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-blue-100 text-blue-600 text-xs">
                  ✓
                </span>
                Vercel
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-blue-100 text-blue-600 text-xs">
                  ✓
                </span>
                Netlify
              </li>
              <li className="flex items-center gap-3 text-lg font-medium text-gray-400">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-800 text-gray-500 text-xs">
                  ○
                </span>
                Fly.io & Railway (Coming soon)
              </li>
            </ul>
          </div>
        </div>
      </section>

      {/* FAQ Section */}
      <section className="w-full max-w-4xl px-4 py-24 mb-10">
        <h2 className="text-3xl font-bold mb-12 text-center">
          Frequently Asked Questions
        </h2>
        <div className="space-y-4">
          <Accordion title="Which LLM models are supported?" icon="cpu">
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed p-4">
              Atlas natively supports Anthropic (Claude), OpenAI, Google Gemini,
              Mistral, Groq, xAI (Grok), and even local models via Ollama. The
              models are strictly used for the <code>FixCode</code> diagnostic
              phase, meaning you control when and how AI is invoked.
            </p>
          </Accordion>

          <Accordion title="How are Django projects deployed?" icon="database">
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed p-4">
              Atlas features a dedicated, rigorous pre-flight checklist for
              Django. It forces you to drop SQLite for production, checks for
              env-controlled <code>SECRET_KEY</code> and <code>DEBUG</code>,
              verifies <code>WhiteNoiseMiddleware</code>, and auto-provisions a
              secure PostgreSQL database on Render automatically. Django
              deployment is currently supported on Render only.
            </p>
          </Accordion>

          <Accordion
            title="Can I use Atlas in CI/CD pipelines?"
            icon="terminal"
          >
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed p-4">
              Yes. Atlas supports fully non-interactive mode via flags:{" "}
              <code>atlas . --action deploy --provider vercel</code>. Provide
              your LLM and provider API keys as environment variables, and Atlas
              will run without any interactive prompts. tion, checks for
              env-controlled `SECRET_KEY` and `DEBUG`, verifies
              `WhiteNoiseMiddleware`, and auto-provisions a secure PostgreSQL
              database on Render automatically.
            </p>
          </Accordion>
        </div>
      </section>
    </div>
  );
}
