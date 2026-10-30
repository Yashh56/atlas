export function PipelineSteps() {
  const steps = [
    {
      number: "1",
      title: "Analyze & Audit",
      tag: "Deterministic",
      tagColor: "bg-zinc-100 text-zinc-700 border-zinc-300 dark:bg-zinc-800 dark:text-zinc-300 dark:border-zinc-700",
      description:
        "Atlas reads your project structure, detects monorepo workspaces (pnpm, npm, yarn, turborepo), identifies frameworks (Next.js, React, Django, Express, Go), and audits environment variables against .env.example templates—zero configuration required.",
    },
    {
      number: "2",
      title: "Build Locally",
      tag: "Sandboxed",
      tagColor: "bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/20",
      description:
        "Executes your framework build command inside an isolated workspace sandbox. Atlas verifies exit codes, tracks artifact output directories (dist, .next, build), and captures complete compiler logs without polluting your global environment.",
    },
    {
      number: "3",
      title: "Self-Healing Fix (if needed)",
      tag: "Bounded AI",
      tagColor: "bg-purple-500/10 text-purple-700 dark:text-purple-300 border-purple-500/20",
      description:
        "If compilation fails, Atlas routes diagnostics to your configured LLM (Claude, OpenAI, Gemini, Ollama). The model produces exact-match patches only. Atlas creates file snapshots and uses SHA-256 hash checks to detect stuck loops, bounded by a strict retry ceiling.",
    },
    {
      number: "4",
      title: "Cloud Deploy",
      tag: "Automated",
      tagColor: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/20",
      description:
        "After a green local build, Atlas deploys directly to your chosen provider (Vercel, Render, or Netlify). For Django on Render, it automatically provisions a free-tier PostgreSQL database and injects the connection credentials.",
    },
    {
      number: "5",
      title: "Health Check & Rollback",
      tag: "Continuous",
      tagColor: "bg-sky-500/10 text-sky-700 dark:text-sky-300 border-sky-500/20",
      description:
        "Atlas executes automated synthetic HTTP health checks against the live deployment URL with exponential-backoff retries. If the health check fails, Atlas prompts or automatically reverts to the previous verified deployment.",
    },
  ];

  return (
    <section className="w-full max-w-4xl mx-auto px-4 py-20 relative z-10 border-t border-zinc-200 dark:border-zinc-800/80">
      <div className="flex flex-col mb-14">
        <span className="text-xs font-semibold uppercase tracking-wider text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-3 py-1 rounded-full border border-emerald-500/20 w-fit mb-3">
          Lifecycle Flow
        </span>
        <h2 className="text-3xl sm:text-4xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50">
          How Atlas Works
        </h2>
        <p className="text-zinc-600 dark:text-zinc-400 text-base sm:text-lg mt-2">
          A deterministic pipeline with one bounded AI step — not a free-roaming agent.
        </p>
      </div>

      <div className="relative">
        {/* Step List */}
        <div className="space-y-6 sm:space-y-8">
          {steps.map((step, idx) => (
            <div key={step.number} className="flex group">
              {/* Number and Connector */}
              <div className="flex flex-col items-center mr-5 sm:mr-6">
                <div className="w-9 h-9 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-950 dark:text-zinc-100 flex items-center justify-center font-bold text-sm z-10 shadow-sm group-hover:border-zinc-400 dark:group-hover:border-zinc-500 transition-colors">
                  {step.number}
                </div>
                {idx < steps.length - 1 && (
                  <div className="w-px h-full bg-zinc-200 dark:bg-zinc-800 my-2" />
                )}
              </div>

              {/* Step Content */}
              <div className="pb-6 pt-0.5 flex-1">
                <div className="flex flex-wrap items-center gap-2 mb-2">
                  <h3 className="text-lg sm:text-xl font-semibold text-zinc-950 dark:text-zinc-50">
                    {step.title}
                  </h3>
                  <span
                    className={`inline-block text-[11px] font-mono font-medium px-2 py-0.5 rounded-full border ${step.tagColor}`}
                  >
                    {step.tag}
                  </span>
                </div>
                <p className="text-zinc-600 dark:text-zinc-400 text-sm sm:text-base leading-relaxed">
                  {step.description}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
