export function PipelineSteps() {
  return (
    <section className="w-full max-w-4xl mx-auto px-4 py-20 relative z-10 border-t border-zinc-200 dark:border-zinc-800">
      <div className="flex flex-col mb-12">
        <h2 className="text-3xl font-bold tracking-tight mb-4 text-zinc-950 dark:text-zinc-50">
          How Atlas Works
        </h2>
        <p className="text-zinc-600 dark:text-zinc-400 text-lg">
          A deterministic pipeline with one bounded AI step — not a free-roaming agent.
        </p>
      </div>

      <div className="space-y-0">
        <div className="flex">
          <div className="flex flex-col items-center mr-6">
            <div className="w-8 h-8 rounded-full border border-zinc-200 bg-white text-zinc-900 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-100 flex items-center justify-center font-bold text-sm z-10 shadow-sm">1</div>
            <div className="w-px h-full bg-zinc-200 dark:bg-zinc-800 my-2"></div>
          </div>
          <div className="pb-10 pt-1">
            <h3 className="text-xl font-semibold mb-2 text-zinc-950 dark:text-zinc-50">Analyze</h3>
            <p className="text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Atlas reads your project structure, detects your framework (Next.js, React, Django, Express, Go), identifies your package manager, and resolves the correct build and start commands — zero config required.
            </p>
          </div>
        </div>

        <div className="flex">
          <div className="flex flex-col items-center mr-6">
            <div className="w-8 h-8 rounded-full border border-zinc-200 bg-white text-zinc-900 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-100 flex items-center justify-center font-bold text-sm z-10 shadow-sm">2</div>
            <div className="w-px h-full bg-zinc-200 dark:bg-zinc-800 my-2"></div>
          </div>
          <div className="pb-10 pt-1">
            <h3 className="text-xl font-semibold mb-2 text-zinc-950 dark:text-zinc-50">Build Locally</h3>
            <p className="text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Runs your build command in a sandboxed workspace. If it succeeds, Atlas proceeds directly to deployment.
            </p>
          </div>
        </div>

        <div className="flex">
          <div className="flex flex-col items-center mr-6">
            <div className="w-8 h-8 rounded-full border border-zinc-200 bg-white text-zinc-900 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-100 flex items-center justify-center font-bold text-sm z-10 shadow-sm">3</div>
            <div className="w-px h-full bg-zinc-200 dark:bg-zinc-800 my-2"></div>
          </div>
          <div className="pb-10 pt-1">
            <h3 className="text-xl font-semibold mb-2 text-zinc-950 dark:text-zinc-50">Fix (if needed)</h3>
            <p className="text-zinc-600 dark:text-zinc-400 leading-relaxed">
              If the build fails, Atlas captures the full error output and sends it to your configured LLM (Claude, OpenAI, Gemini, etc.). The model proposes a fix as an exact-match string replacement — no full-file rewrites. Retries are strictly bounded.
            </p>
          </div>
        </div>

        <div className="flex">
          <div className="flex flex-col items-center mr-6">
            <div className="w-8 h-8 rounded-full border border-zinc-200 bg-white text-zinc-900 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-100 flex items-center justify-center font-bold text-sm z-10 shadow-sm">4</div>
            <div className="w-px h-full bg-zinc-200 dark:bg-zinc-800 my-2"></div>
          </div>
          <div className="pb-10 pt-1">
            <h3 className="text-xl font-semibold mb-2 text-zinc-950 dark:text-zinc-50">Deploy</h3>
            <p className="text-zinc-600 dark:text-zinc-400 leading-relaxed">
              After a successful build, Atlas deploys to your chosen provider (Vercel, Render, or Netlify). For Django on Render, it auto-provisions a free-tier PostgreSQL database and wires the connection string.
            </p>
          </div>
        </div>

        <div className="flex">
          <div className="flex flex-col items-center mr-6">
            <div className="w-8 h-8 rounded-full border border-zinc-200 bg-white text-zinc-900 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-100 flex items-center justify-center font-bold text-sm z-10 shadow-sm">5</div>
          </div>
          <div className="pb-4 pt-1">
            <h3 className="text-xl font-semibold mb-2 text-zinc-950 dark:text-zinc-50">Health Check & Rollback</h3>
            <p className="text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Atlas runs an HTTP health check against your live URL with exponential-backoff retries. If the check fails and a prior healthy deployment exists, you are prompted to rollback.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}
