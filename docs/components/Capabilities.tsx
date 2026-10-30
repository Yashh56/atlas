import {
  Activity,
  Cpu,
  Database,
  KeyRound,
  Layers,
  Sparkles,
} from "lucide-react";

export function Capabilities() {
  const capabilities = [
    {
      icon: Layers,
      title: "Monorepos & Workspaces",
      description:
        "Native discovery for pnpm workspaces, npm, yarn, and Turborepo. Target specific applications via --workspace or --dir without losing monorepo root context or package resolution.",
      tags: ["pnpm", "turborepo", "--workspace", "yarn"],
      featured: false,
    },
    {
      icon: KeyRound,
      title: "Pre-Flight Secret Auditing",
      description:
        "Atlas audits declared .env.example templates against OS Keychain credentials and process environment before touching the cloud. Flag missing production keys with --strict-env before code ships.",
      tags: ["atlas audit", "--strict-env", "OS Keychain"],
      featured: false,
    },
    {
      icon: Cpu,
      title: "Self-Healing Build Engine",
      description:
        "Deterministic pipeline with one tightly bounded AI diagnostic step. FixCode diagnoses compiler errors and executes exact-match string replacements—never full-file rewrites.",
      tags: ["Exact-Match", "SHA-256 Guard", "Sandboxed"],
      featured: false,
    },
    {
      icon: Sparkles,
      title: "Multi-LLM Provider Support",
      description:
        "Swap between Anthropic Claude 3.5, OpenAI GPT-4o, Google Gemini 2.5, Mistral, Groq, xAI, or local offline Ollama models for the FixCode step. Zero model lock-in.",
      tags: ["Claude 3.5", "GPT-4o", "Gemini 2.5", "Ollama"],
      featured: false,
    },
    {
      icon: Database,
      title: "Render Postgres Provisioning",
      description:
        "Dedicated infrastructure setup for Django. Atlas provisions free-tier PostgreSQL instances on Render, binds DATABASE_URL automatically, and enforces production database settings.",
      tags: ["PostgreSQL", "Django", "Render API", "Auto-Wire"],
      featured: false,
    },
    {
      icon: Activity,
      title: "Health Checks & Rollback",
      description:
        "Post-deployment synthetic HTTP probes verify live application uptime. If the check fails, Atlas protects your production environment with instant rollback triggers.",
      tags: ["Synthetic Probe", "Exponential Backoff", "Rollback"],
      featured: false,
    },
  ];

  return (
    <section className="w-full max-w-6xl mx-auto px-4 py-20 relative z-10 border-t border-zinc-200 dark:border-zinc-800/80">
      <div className="flex flex-col mb-12">
        <span className="text-xs font-semibold uppercase tracking-wider text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-3 py-1 rounded-full border border-emerald-500/20 w-fit mb-3">
          Architecture &amp; Features
        </span>
        <h2 className="text-3xl sm:text-4xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50">
          Built-in Capabilities
        </h2>
        <p className="text-zinc-600 dark:text-zinc-400 text-base sm:text-lg mt-2 max-w-2xl">
          Designed for modern developer ergonomics, multi-repo setups, and rock-solid production deploys.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {capabilities.map((item) => {
          const Icon = item.icon;
          return (
            <div
              key={item.title}
              className="flex flex-col justify-between rounded-2xl border border-zinc-200 dark:border-zinc-800/80 bg-white dark:bg-zinc-900/50 p-6 shadow-sm hover:border-zinc-300 dark:hover:border-zinc-700 transition-all duration-200 group"
            >
              <div>
                <div className="w-10 h-10 rounded-xl bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mb-4 border border-zinc-200 dark:border-zinc-700/60 text-zinc-900 dark:text-zinc-100 group-hover:scale-105 transition-transform">
                  <Icon className="w-5 h-5" />
                </div>
                <h3 className="font-semibold text-zinc-950 dark:text-zinc-50 text-lg mb-2">
                  {item.title}
                </h3>
                <p className="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed mb-6">
                  {item.description}
                </p>
              </div>

              <div className="flex flex-wrap gap-1.5 pt-2 border-t border-zinc-100 dark:border-zinc-800/60">
                {item.tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-block text-[11px] font-mono px-2 py-0.5 rounded-md bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-300 border border-zinc-200/80 dark:border-zinc-700/50"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
