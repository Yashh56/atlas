import { useState } from "react";
import { Check, Copy, Play, ShieldAlert, Stethoscope, Terminal } from "lucide-react";

type TabKey = "deploy" | "audit" | "doctor";

export function TerminalExample() {
  const [activeTab, setActiveTab] = useState<TabKey>("deploy");
  const [copied, setCopied] = useState(false);

  const tabCommands: Record<TabKey, string> = {
    deploy: "atlas . --provider vercel",
    audit: "atlas audit --workspace apps/web",
    doctor: "atlas doctor",
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(tabCommands[activeTab]);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section className="w-full max-w-4xl px-4 py-10 relative z-10 mx-auto">
      <div className="rounded-2xl overflow-hidden border border-zinc-200 dark:border-zinc-800 bg-zinc-950 shadow-2xl backdrop-blur-sm">
        {/* Terminal Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between px-4 py-3 border-b border-zinc-800 bg-zinc-900/70 gap-2">
          <div className="flex items-center gap-3">
            <div className="flex space-x-2">
              <div className="w-3 h-3 rounded-full bg-rose-500/80"></div>
              <div className="w-3 h-3 rounded-full bg-amber-500/80"></div>
              <div className="w-3 h-3 rounded-full bg-emerald-500/80"></div>
            </div>
            <div className="text-xs text-zinc-400 font-mono flex items-center gap-1.5 ml-2">
              <Terminal className="w-3.5 h-3.5 text-zinc-500" />
              <span>atlas-terminal ~ v0.4.0</span>
            </div>
          </div>

          {/* Interactive Mode Tabs */}
          <div className="flex items-center space-x-1 bg-zinc-950/80 p-1 rounded-lg border border-zinc-800/80 self-start sm:self-auto">
            <button
              type="button"
              onClick={() => setActiveTab("deploy")}
              className={`flex items-center gap-1.5 px-2.5 py-1 text-xs font-mono rounded-md transition-all ${
                activeTab === "deploy"
                  ? "bg-zinc-800 text-zinc-100 shadow-sm font-semibold"
                  : "text-zinc-400 hover:text-zinc-200"
              }`}
            >
              <Play className="w-3 h-3 text-emerald-400" />
              <span>atlas .</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("audit")}
              className={`flex items-center gap-1.5 px-2.5 py-1 text-xs font-mono rounded-md transition-all ${
                activeTab === "audit"
                  ? "bg-zinc-800 text-zinc-100 shadow-sm font-semibold"
                  : "text-zinc-400 hover:text-zinc-200"
              }`}
            >
              <ShieldAlert className="w-3 h-3 text-cyan-400" />
              <span>atlas audit</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("doctor")}
              className={`flex items-center gap-1.5 px-2.5 py-1 text-xs font-mono rounded-md transition-all ${
                activeTab === "doctor"
                  ? "bg-zinc-800 text-zinc-100 shadow-sm font-semibold"
                  : "text-zinc-400 hover:text-zinc-200"
              }`}
            >
              <Stethoscope className="w-3 h-3 text-amber-400" />
              <span>atlas doctor</span>
            </button>
          </div>
        </div>

        {/* Terminal Body */}
        <div className="p-5 sm:p-6 text-xs sm:text-sm font-mono text-zinc-300 overflow-x-auto text-left leading-relaxed relative">
          <button
            type="button"
            onClick={handleCopy}
            className="absolute top-4 right-4 p-1.5 rounded-md bg-zinc-900/90 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 border border-zinc-800 transition-colors"
            title="Copy command"
            aria-label="Copy command"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
          </button>

          {/* TAB 1: atlas . (Deploy & Self-Heal) */}
          {activeTab === "deploy" && (
            <div className="space-y-1.5">
              <div className="flex items-center gap-2 text-zinc-100 font-semibold mb-3">
                <span className="text-emerald-400">➜</span>
                <span className="text-cyan-400">~/projects/acme-monorepo</span>
                <span className="text-zinc-500">$</span>
                <span>atlas . --provider vercel</span>
              </div>
              <div className="text-zinc-500 text-xs pb-1">◈ Atlas v0.4.0 — Autonomous Self-Healing Pipeline</div>

              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-zinc-800 text-zinc-300 border border-zinc-700/60">INIT</span>
                <span className="text-zinc-300">Monorepo detected: <span className="text-zinc-100">pnpm-workspace.yaml</span> (apps/web, packages/ui)</span>
              </div>
              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-blue-500/10 text-blue-400 border border-blue-500/20">DISC</span>
                <span className="text-zinc-300">Framework detected: <span className="text-blue-300 font-medium">Next.js 15.1.0</span> in <span className="text-zinc-200">apps/web</span></span>
              </div>
              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-zinc-800 text-zinc-300 border border-zinc-700/60">BUILD</span>
                <span className="text-zinc-400">Running sandboxed build: <code className="text-zinc-200">pnpm --filter apps/web build</code></span>
              </div>

              <div className="my-2 p-2.5 rounded bg-rose-950/20 border border-rose-900/40 text-rose-300/90 text-xs">
                <div className="font-semibold text-rose-400 flex items-center gap-1.5">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-rose-500/20 text-rose-300">FAIL</span>
                  <span>Exit status 1 — Compilation Error</span>
                </div>
                <div className="mt-1 text-zinc-400">
                  <span>apps/web/src/lib/metrics.ts:42:15</span><br />
                  <span className="text-rose-300">TS2322: Type &apos;string | undefined&apos; is not assignable to type &apos;string&apos;.</span>
                </div>
              </div>

              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">DIAG</span>
                <span className="text-amber-200/90">Diagnosing failure with <span className="text-amber-300 font-semibold">Claude 3.5 Sonnet</span> (attempt 1/3)...</span>
              </div>
              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-purple-500/10 text-purple-400 border border-purple-500/20">SNAP</span>
                <span className="text-zinc-400">Created file snapshot: <code className="text-zinc-300">.atlas/snapshots/metrics.ts.bak</code></span>
              </div>
              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">FIX</span>
                <span className="text-zinc-300">Generated exact-match patch <span className="text-zinc-400">(-1, +1 lines)</span></span>
              </div>
              <div className="pl-6 text-xs text-zinc-400 font-mono py-1">
                <span className="text-rose-400 block">- const apiKey: string = process.env.METRICS_KEY;</span>
                <span className="text-emerald-400 block">+ const apiKey: string = process.env.METRICS_KEY ?? &quot;dev-fallback&quot;;</span>
              </div>
              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">PASS</span>
                <span className="text-emerald-300">Rebuilt sandboxed workspace successfully in 2.84s</span>
              </div>
              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-blue-500/10 text-blue-400 border border-blue-500/20">SHIP</span>
                <span className="text-zinc-300">Shipping bundle to Vercel production deployment...</span>
              </div>
              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">PASS</span>
                <span className="text-emerald-300">Synthetic HTTP health check passed (200 OK in 142ms)</span>
              </div>

              <div className="mt-4 pt-3 border-t border-zinc-800/80 flex items-center justify-between flex-wrap gap-2">
                <div className="flex items-center gap-2 text-zinc-100 font-semibold">
                  <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-sky-500/10 text-sky-400 border border-sky-500/20">LIVE</span>
                  <span>Ready at:</span>
                  <a
                    href="https://web-atlas-demo.vercel.app"
                    target="_blank"
                    rel="noreferrer"
                    className="text-sky-400 hover:text-sky-300 underline underline-offset-2"
                  >
                    https://web-atlas-demo.vercel.app
                  </a>
                </div>
                <span className="text-zinc-500 text-xs">Duration: 7.2s</span>
              </div>
            </div>
          )}

          {/* TAB 2: atlas audit (Pre-Flight Secret Audit) */}
          {activeTab === "audit" && (
            <div className="space-y-1.5">
              <div className="flex items-center gap-2 text-zinc-100 font-semibold mb-3">
                <span className="text-emerald-400">➜</span>
                <span className="text-cyan-400">~/projects/acme-monorepo</span>
                <span className="text-zinc-500">$</span>
                <span>atlas audit --workspace apps/web</span>
              </div>
              <div className="text-zinc-500 text-xs pb-1">◈ Atlas v0.4.0 — Pre-Flight Secret &amp; Environment Auditor</div>

              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">SCAN</span>
                <span className="text-zinc-300">Discovered templates: <code className="text-zinc-200">.env.example</code>, <code className="text-zinc-200">apps/web/.env.example</code> (7 keys)</span>
              </div>
              <div className="flex items-start gap-2">
                <span className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold bg-zinc-800 text-zinc-300 border border-zinc-700/60">PROBE</span>
                <span className="text-zinc-400">Cross-checking OS Keychain (credentials.Store) and runtime environment...</span>
              </div>

              <div className="mt-2 space-y-1 font-mono text-xs pl-2">
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="font-semibold text-zinc-200 w-44">DATABASE_URL</span>
                  <span className="text-zinc-400">→ Present in Keychain</span>
                  <span className="text-zinc-600">(postgresql://***:***@aws.neon.tech/prod)</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="font-semibold text-zinc-200 w-44">NEXT_PUBLIC_APP_URL</span>
                  <span className="text-zinc-400">→ Present in process env</span>
                  <span className="text-zinc-600">(https://acme.org)</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="font-semibold text-zinc-200 w-44">AUTH_SECRET</span>
                  <span className="text-zinc-400">→ Present in Keychain</span>
                  <span className="text-zinc-600">(32-byte cryptographically secure token)</span>
                </div>
                <div className="flex items-center gap-2 text-amber-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-amber-500/10 text-amber-400 border border-amber-500/20 font-semibold">WARN</span>
                  <span className="font-semibold text-amber-200 w-44">STRIPE_WEBHOOK_SECRET</span>
                  <span className="text-amber-400">→ Missing from environment &amp; Keychain</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-400">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-zinc-800 text-zinc-400 border border-zinc-700/60 font-semibold">INFO</span>
                  <span className="font-semibold text-zinc-300 w-44">POSTHOG_API_KEY</span>
                  <span className="text-zinc-500">→ Optional variable omitted</span>
                </div>
              </div>

              <div className="mt-3 p-2.5 rounded bg-cyan-950/20 border border-cyan-800/40 text-cyan-200/90 text-xs">
                <div className="flex items-center gap-1.5 font-semibold text-cyan-400">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-cyan-500/20 text-cyan-300">AUDIT</span>
                  <span>Safety Invariant Verified: Zero Plaintext Secrets</span>
                </div>
                <p className="mt-1 text-zinc-400">
                  Atlas audited 7 variables. Zero secrets were logged or stored in .atlas/ state files.
                </p>
              </div>

              <div className="text-xs text-amber-400/90 pt-1">
                Advice: run <code className="text-amber-300 font-semibold">atlas credentials set STRIPE_WEBHOOK_SECRET</code> before triggering production deploy.
              </div>
            </div>
          )}

          {/* TAB 3: atlas doctor (System Diagnostics) */}
          {activeTab === "doctor" && (
            <div className="space-y-1.5">
              <div className="flex items-center gap-2 text-zinc-100 font-semibold mb-3">
                <span className="text-emerald-400">➜</span>
                <span className="text-cyan-400">~/projects</span>
                <span className="text-zinc-500">$</span>
                <span>atlas doctor</span>
              </div>
              <div className="text-zinc-500 text-xs pb-1">◈ Atlas v0.4.0 — Environment &amp; Provider Diagnostics</div>

              <div className="text-xs font-semibold text-zinc-400 uppercase tracking-wider pt-1">Runtimes &amp; Tools</div>
              <div className="space-y-1 pl-2">
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">Node.js:</span>
                  <span className="text-zinc-200">v22.11.0 (package manager: pnpm v9.14.2)</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">Python:</span>
                  <span className="text-zinc-200">Python 3.12.3 (via py launcher, venv supported)</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">Go:</span>
                  <span className="text-zinc-200">go1.23.2 windows/amd64</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">Git:</span>
                  <span className="text-zinc-200">git 2.47.0 (clean working tree, origin/main)</span>
                </div>
              </div>

              <div className="text-xs font-semibold text-zinc-400 uppercase tracking-wider pt-2">Cloud Providers</div>
              <div className="space-y-1 pl-2">
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">Vercel:</span>
                  <span className="text-zinc-200">Token valid (team: &quot;atlas-eng&quot;, scope: deploy)</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">Render:</span>
                  <span className="text-zinc-200">API Key active (web services &amp; PostgreSQL ready)</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">Netlify:</span>
                  <span className="text-zinc-200">Personal access token configured</span>
                </div>
              </div>

              <div className="text-xs font-semibold text-zinc-400 uppercase tracking-wider pt-2">LLM Diagnostic Models</div>
              <div className="space-y-1 pl-2">
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">Anthropic:</span>
                  <span className="text-zinc-200">Claude 3.5 Sonnet / Haiku active (ping: 198ms)</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">PASS</span>
                  <span className="w-24 text-zinc-400">OpenAI:</span>
                  <span className="text-zinc-200">GPT-4o active (ping: 165ms)</span>
                </div>
                <div className="flex items-center gap-2 text-zinc-300">
                  <span className="inline-block px-1.5 py-0.2 rounded text-[10px] bg-blue-500/10 text-blue-400 border border-blue-500/20 font-semibold">INFO</span>
                  <span className="w-24 text-zinc-400">Ollama:</span>
                  <span className="text-zinc-200">Local daemon active at http://127.0.0.1:11434</span>
                </div>
              </div>

              <div className="mt-3 pt-2 text-emerald-400 text-xs font-semibold flex items-center gap-2 border-t border-zinc-800">
                <span className="text-emerald-400">✓</span>
                <span>All 10 doctor checks passed. System ready for autonomous operations.</span>
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
