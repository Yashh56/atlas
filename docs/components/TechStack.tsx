import { CheckCircle2, Clock, Cloud, Globe, Server } from "lucide-react";

export function TechStack() {
  const webFrameworks = [
    { name: "Next.js", icon: "devicon-nextjs-plain colored dark:invert", status: "Supported" },
    { name: "React", icon: "devicon-react-original colored", status: "Supported" },
    { name: "Vite", icon: "devicon-vitejs-plain colored", status: "Supported" },
    { name: "Remix", icon: "devicon-remix-original colored", status: "Supported" },
    { name: "Astro", icon: "devicon-astro-plain colored", status: "Supported" },
    { name: "SvelteKit", icon: "devicon-svelte-plain colored", status: "Supported" },
    { name: "Nuxt", icon: "devicon-nuxtjs-plain colored", status: "Supported" },
  ];

  const backendRuntimes = [
    { name: "Node.js", icon: "devicon-nodejs-plain colored", status: "Supported" },
    { name: "Express", icon: "devicon-express-original colored dark:invert", status: "Supported" },
    { name: "Django", icon: "devicon-django-plain colored", status: "Supported (Render)" },
    { name: "Go", icon: "devicon-go-plain colored", status: "Supported" },
    { name: "Cargo / Rust", icon: "devicon-rust-original colored", status: "Supported" },
    { name: "Python", icon: "devicon-python-plain colored", status: "Supported" },
  ];

  const providers = [
    {
      name: "Render",
      desc: "Full web services & automated PostgreSQL provisioning",
      ready: true,
    },
    {
      name: "Vercel",
      desc: "Frontend, static edge, and Next.js serverless runtimes",
      ready: true,
    },
    {
      name: "Netlify",
      desc: "Edge-hosted websites and serverless functions",
      ready: true,
    },
    {
      name: "Fly.io & Railway",
      desc: "Containerized deployments and microservices",
      ready: false,
    },
  ];

  return (
    <section className="w-full max-w-6xl mx-auto px-4 py-20 border-t border-zinc-200 dark:border-zinc-800/80">
      <div className="flex flex-col mb-12">
        <span className="text-xs font-semibold uppercase tracking-wider text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-3 py-1 rounded-full border border-emerald-500/20 w-fit mb-3">
          Ecosystem &amp; Targets
        </span>
        <h2 className="text-3xl sm:text-4xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50">
          Supported Tech Stack
        </h2>
        <p className="text-zinc-600 dark:text-zinc-400 text-base sm:text-lg mt-2 max-w-2xl">
          Atlas auto-detects frameworks, package managers, and build artifacts without manual configuration.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Category 1: Web Frameworks */}
        <div className="rounded-2xl border border-zinc-200 dark:border-zinc-800/80 bg-white dark:bg-zinc-900/50 p-6 flex flex-col justify-between">
          <div>
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-600 dark:text-blue-400">
                <Globe className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-semibold text-zinc-950 dark:text-zinc-50 text-base">
                  Web Frameworks
                </h3>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  SPA, SSR, and hybrid fullstack
                </p>
              </div>
            </div>

            <div className="flex flex-wrap gap-2 mt-4">
              {webFrameworks.map((fw) => (
                <div
                  key={fw.name}
                  className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-zinc-50 dark:bg-zinc-800/80 border border-zinc-200/90 dark:border-zinc-700/60 text-xs font-medium text-zinc-800 dark:text-zinc-200 shadow-sm"
                >
                  <i className={`${fw.icon} text-base shrink-0`} />
                  <span>{fw.name}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Category 2: Backend Runtimes */}
        <div className="rounded-2xl border border-zinc-200 dark:border-zinc-800/80 bg-white dark:bg-zinc-900/50 p-6 flex flex-col justify-between">
          <div>
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                <Server className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-semibold text-zinc-950 dark:text-zinc-50 text-base">
                  Backend &amp; Languages
                </h3>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  APIs, microservices, and databases
                </p>
              </div>
            </div>

            <div className="flex flex-wrap gap-2 mt-4">
              {backendRuntimes.map((rt) => (
                <div
                  key={rt.name}
                  className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-zinc-50 dark:bg-zinc-800/80 border border-zinc-200/90 dark:border-zinc-700/60 text-xs font-medium text-zinc-800 dark:text-zinc-200 shadow-sm"
                >
                  <i className={`${rt.icon} text-base shrink-0`} />
                  <span>{rt.name}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Category 3: Cloud Providers */}
        <div className="rounded-2xl border border-zinc-200 dark:border-zinc-800/80 bg-white dark:bg-zinc-900/50 p-6 flex flex-col justify-between">
          <div>
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-600 dark:text-purple-400">
                <Cloud className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-semibold text-zinc-950 dark:text-zinc-50 text-base">
                  Cloud Providers
                </h3>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Native deployment adapters
                </p>
              </div>
            </div>

            <div className="space-y-2 mt-4">
              {providers.map((p) => (
                <div
                  key={p.name}
                  className="p-2.5 rounded-lg bg-zinc-50 dark:bg-zinc-800/80 border border-zinc-200/80 dark:border-zinc-700/60 flex items-start gap-2.5"
                >
                  {p.ready ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-500 mt-0.5 shrink-0" />
                  ) : (
                    <Clock className="w-4 h-4 text-zinc-400 dark:text-zinc-500 mt-0.5 shrink-0" />
                  )}
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-semibold text-zinc-900 dark:text-zinc-100">
                        {p.name}
                      </span>
                      {!p.ready && (
                        <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-zinc-200 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-300">
                          Coming Soon
                        </span>
                      )}
                    </div>
                    <p className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 leading-snug">
                      {p.desc}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
