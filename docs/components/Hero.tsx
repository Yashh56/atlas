import { Link, Rocket } from "lucide-react";
import { useState } from "react";

export function InstallCommand() {
  const [os, setOs] = useState<"mac" | "windows" | "go">("mac");
  const [copied, setCopied] = useState(false);

  const commands = {
    mac: "curl -fsSL https://atlas.yashh56.me/install.sh | sh",
    windows: "iwr -useb https://atlas.yashh56.me/install.ps1 | iex",
    go: "go install github.com/Yashh56/atlas/cmd/atlas@latest"
  };

  const copyToClipboard = () => {
    navigator.clipboard.writeText(commands[os]);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="w-full max-w-2xl mx-auto mt-12 flex flex-col items-center">
      <div className="flex space-x-1 bg-zinc-100/80 dark:bg-zinc-900/80 p-1 rounded-lg mb-4 border border-zinc-200 dark:border-zinc-800 backdrop-blur-sm">
        <button
          onClick={() => setOs("mac")}
          className={`px-4 py-1.5 text-sm font-medium rounded-md transition-all ${os === "mac" ? "bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 shadow-sm" : "text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"}`}
        >
          macOS / Linux
        </button>
        <button
          onClick={() => setOs("windows")}
          className={`px-4 py-1.5 text-sm font-medium rounded-md transition-all ${os === "windows" ? "bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 shadow-sm" : "text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"}`}
        >
          Windows
        </button>
        <button
          onClick={() => setOs("go")}
          className={`px-4 py-1.5 text-sm font-medium rounded-md transition-all ${os === "go" ? "bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 shadow-sm" : "text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"}`}
        >
          Go
        </button>
      </div>

      <div className="relative w-full group">
        <pre className="bg-zinc-950 border border-zinc-800 rounded-xl p-4 text-sm text-zinc-50 font-mono text-left shadow-2xl whitespace-pre-wrap break-all">
          <code>{commands[os]}</code>
        </pre>
        <button
          onClick={copyToClipboard}
          className="absolute right-3 top-3 p-1.5 rounded-md bg-zinc-800/80 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-700 transition-all opacity-0 group-hover:opacity-100"
          aria-label="Copy code"
        >
          {copied ? (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
          ) : (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
          )}
        </button>
      </div>
    </div>
  );
}

export function Hero() {
  return (
    <section className="w-full pt-32 pb-16 flex flex-col items-center text-center px-4 relative z-10">
      <a href="https://github.com/Yashh56/atlas" target="_blank" rel="noreferrer" className="inline-flex items-center rounded-full border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/50 px-3 py-1 text-sm font-medium text-zinc-900 dark:text-zinc-200 mb-8 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors shadow-sm">
        <svg className="mr-2 h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" /><path d="M9 18c-4.51 2-5-2-7-2" /></svg>
        Star on GitHub
      </a>

      <h1 className="text-6xl md:text-8xl font-extrabold tracking-tight mb-2 text-zinc-950 dark:text-white max-w-4xl">
        Atlas
      </h1>

      <h2 className="text-3xl md:text-5xl font-bold tracking-tight mb-6 text-transparent bg-clip-text bg-gradient-to-r from-zinc-900 to-zinc-500 dark:from-zinc-100 dark:to-zinc-500 max-w-4xl">
        The autonomous, self-healing <br className="hidden md:block" />
        deployment pipeline.
      </h2>

      <h3 className="text-lg md:text-xl font-normal text-zinc-600 dark:text-zinc-400 max-w-2xl mb-8 leading-relaxed">
        Atlas analyzes your project, executes local builds, diagnoses failures with LLMs, and safely ships to the cloud. Zero configuration required.
      </h3>

      <div className="flex flex-col sm:flex-row gap-4 w-full sm:w-auto mb-8">
        <a href="/docs" className="inline-flex h-11 items-center justify-center rounded-md bg-zinc-900 px-8 text-sm font-medium text-zinc-50 shadow transition-colors hover:bg-zinc-900/90 dark:bg-zinc-50 dark:text-zinc-900 dark:hover:bg-zinc-50/90 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-zinc-950 disabled:pointer-events-none disabled:opacity-50">
          Get Started <Rocket className="ml-2 h-5 w-5" />
        </a>
        <a href="https://github.com/Yashh56/atlas" target="_blank" rel="noreferrer" className="inline-flex h-11 items-center justify-center rounded-md border border-zinc-200 bg-white px-8 text-sm font-medium shadow-sm transition-colors hover:bg-zinc-100 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:bg-zinc-800 dark:hover:text-zinc-50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-zinc-950 disabled:pointer-events-none disabled:opacity-50">
          View GitHub <i className="ml-4 text-xl devicon-github-original" />
        </a>
      </div>

      <InstallCommand />
    </section>
  );
}
