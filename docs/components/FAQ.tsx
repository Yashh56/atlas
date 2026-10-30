import { useState } from "react";

function FAQItem({ title, children }: { title: string; children: React.ReactNode }) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="border-b border-zinc-200 dark:border-zinc-800/80">
      <button
        type="button"
        className="flex flex-1 items-center justify-between py-5 font-medium transition-all hover:text-zinc-600 dark:hover:text-zinc-300 text-left w-full text-zinc-950 dark:text-zinc-50 text-base sm:text-lg"
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
      >
        <span>{title}</span>
        <svg
          className={`h-4 w-4 shrink-0 transition-transform duration-200 ml-4 ${isOpen ? "rotate-180" : ""}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {isOpen && (
        <div className="pb-5 pt-0 text-sm sm:text-base text-zinc-600 dark:text-zinc-400 leading-relaxed space-y-2">
          {children}
        </div>
      )}
    </div>
  );
}

export function FAQ() {
  return (
    <section className="w-full max-w-3xl mx-auto px-4 py-20 border-t border-zinc-200 dark:border-zinc-800/80 mb-10">
      <div className="flex flex-col mb-10 text-center">
        <span className="text-xs font-semibold uppercase tracking-wider text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-3 py-1 rounded-full border border-emerald-500/20 w-fit mx-auto mb-3">
          Got Questions?
        </span>
        <h2 className="text-3xl sm:text-4xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50">
          Frequently Asked Questions
        </h2>
      </div>

      <div className="divide-y divide-transparent">
        <FAQItem title="Which LLM models and providers are supported?">
          <p>
            Atlas natively supports Anthropic (Claude 3.5 Sonnet &amp; Haiku), OpenAI (GPT-4o), Google Gemini 2.5, Mistral, Groq, xAI (Grok), and local offline models via Ollama.
          </p>
          <p>
            Models are strictly invoked for the single, tightly bounded <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">FixCode</code> diagnostic phase. Atlas defaults to deterministic Go logic for all builds, environment resolution, and provider orchestration.
          </p>
        </FAQItem>

        <FAQItem title="How does monorepo and workspace support work?">
          <p>
            Atlas automatically discovers workspace configurations from <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">pnpm-workspace.yaml</code>, Turborepo configs, or root <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">package.json</code> workspaces.
          </p>
          <p>
            Use <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">--workspace &lt;name&gt;</code> or <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">--dir &lt;path&gt;</code> to isolate individual applications while retaining shared dependencies and root lockfiles.
          </p>
        </FAQItem>

        <FAQItem title="How does pre-flight secret auditing keep secrets private?">
          <p>
            The <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">atlas audit</code> command parses declared variables in your <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">.env.example</code> templates and cross-references them against your OS Keychain (<code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">credentials.Store</code>) and runtime environment.
          </p>
          <p>
            Under our strict security invariants, secret values are never printed to terminal output, never written to disk, and never stored in <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">.atlas/</code> session files.
          </p>
        </FAQItem>

        <FAQItem title="How are Django projects and databases handled on Render?">
          <p>
            Atlas runs a production readiness check for Django: it enforces moving away from local SQLite, verifies <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">SECRET_KEY</code> and <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">DEBUG=False</code>, and validates static asset serving via WhiteNoise.
          </p>
          <p>
            On Render, Atlas automatically provisions a managed free-tier PostgreSQL instance and binds the resulting <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">DATABASE_URL</code> directly into your web service environment.
          </p>
        </FAQItem>

        <FAQItem title="Can I use Atlas in CI/CD pipelines without prompts?">
          <p>
            Yes. Atlas supports full headless execution using flags: <code className="text-xs font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">atlas . --action deploy --provider vercel --yes</code>.
          </p>
          <p>
            When supplied with provider credentials via environment variables in GitHub Actions, GitLab CI, or CircleCI, Atlas runs end-to-end builds and deployments without interactive prompts.
          </p>
        </FAQItem>
      </div>
    </section>
  );
}
