import { KeyRound, RotateCcw, ShieldCheck } from "lucide-react";

export function SafetyCallout() {
  const pillars = [
    {
      icon: ShieldCheck,
      title: "Exact-Match Sandboxing & Snapshots",
      description:
        "Atlas never executes full-file rewrites. FixCode produces strictly bounded exact-match patches (old_str → new_str). Automatic file-level snapshots are created before touching code, with guaranteed zero-footprint rollback if the fix fails.",
      badge: "Non-Destructive",
    },
    {
      icon: KeyRound,
      title: "Zero Plaintext Secrets in Keychain",
      description:
        "Provider API keys and sensitive tokens live exclusively in your encrypted OS Keychain (credentials.Store) or runtime environment. No credentials ever appear in .atlas/ session state or stdout logs.",
      badge: "Cryptographic Store",
    },
    {
      icon: RotateCcw,
      title: "Bounded Loops & Instant Rollback",
      description:
        "Every diagnostic iteration has a strict ceiling with SHA-256 stuck loop detection. Automated post-deployment synthetic health checks verify live endpoints, triggering immediate rollback if verification fails.",
      badge: "Deterministic Ceiling",
    },
  ];

  return (
    <section className="w-full max-w-5xl mx-auto px-4 py-16 relative z-10">
      <div className="flex flex-col items-center text-center mb-10">
        <span className="text-xs font-semibold uppercase tracking-wider text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-3 py-1 rounded-full border border-emerald-500/20 mb-3">
          Deterministic Safety Invariants
        </span>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50">
          Safe by design. Guardrails by default.
        </h2>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400 text-sm sm:text-base max-w-2xl">
          Atlas is built on deterministic Go invariants — not an unconstrained autonomous agent.
          Consequential actions always respect hard ceilings and explicit confirmation.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {pillars.map((pillar) => {
          const Icon = pillar.icon;
          return (
            <div
              key={pillar.title}
              className="flex flex-col justify-between rounded-2xl border border-zinc-200 dark:border-zinc-800/90 bg-white dark:bg-zinc-900/60 p-6 shadow-sm hover:shadow-md transition-shadow relative overflow-hidden group"
            >
              <div className="absolute top-0 right-0 h-24 w-24 bg-gradient-to-bl from-zinc-100 to-transparent dark:from-zinc-800/40 dark:to-transparent rounded-bl-full pointer-events-none transition-opacity group-hover:opacity-100 opacity-60" />
              <div>
                <div className="flex items-center justify-between mb-4">
                  <div className="w-11 h-11 rounded-xl bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center border border-zinc-200 dark:border-zinc-700/60 text-zinc-900 dark:text-zinc-100">
                    <Icon className="w-5 h-5" />
                  </div>
                  <span className="text-[11px] font-mono font-medium px-2 py-0.5 rounded-full bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-300 border border-zinc-200 dark:border-zinc-700/50">
                    {pillar.badge}
                  </span>
                </div>
                <h3 className="font-semibold text-zinc-950 dark:text-zinc-50 text-base mb-2">
                  {pillar.title}
                </h3>
                <p className="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed">
                  {pillar.description}
                </p>
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
