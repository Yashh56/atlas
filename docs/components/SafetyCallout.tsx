export function SafetyCallout() {
  return (
    <section className="w-full max-w-4xl mx-auto px-4 py-16 relative z-10">
      <div className="relative w-full rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 [&>svg]:absolute [&>svg]:text-zinc-950 dark:[&>svg]:text-zinc-50 [&>svg]:left-6 [&>svg]:top-6 [&>svg+div]:translate-y-[-3px] [&:has(svg)]:pl-14 bg-white dark:bg-zinc-950 text-zinc-950 dark:text-zinc-50 shadow-sm">
        <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
        <h5 className="mb-2 font-semibold leading-none tracking-tight text-lg">Safe by design</h5>
        <div className="text-sm text-zinc-600 dark:text-zinc-400 [&_p]:leading-relaxed leading-relaxed">
          Atlas never performs full-file rewrites. Every LLM-proposed fix is a sandboxed, exact-match patch — if it can't match precisely, it fails safely rather than guessing. All edits are backed up as file-level snapshots and automatically restored if the retry budget is exhausted. Consequential actions always require explicit confirmation.
        </div>
      </div>
    </section>
  );
}
