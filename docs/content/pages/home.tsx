import { Button, Accordion } from "@umami/shiso/components";

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
      <section className="w-full pt-32 pb-24 flex flex-col items-center text-center px-4 relative z-10">
        <div className="inline-flex items-center gap-2 px-3 py-1 mb-8 text-sm font-medium rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 border border-blue-200 dark:border-blue-800/50">
          <span className="flex h-2 w-2 rounded-full bg-blue-500 animate-pulse"></span>
          Now with Full Django Support
        </div>
        
        <h1 className="text-6xl md:text-8xl font-extrabold tracking-tighter mb-6">
          Meet <span className="bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-teal-400 drop-shadow-sm">Atlas</span>
        </h1>
        
        <p className="text-xl md:text-2xl text-gray-600 dark:text-gray-400 max-w-3xl mb-10 leading-relaxed font-medium">
          The autonomous deployment CLI that analyzes your project, executes local builds, 
          diagnoses failures with LLMs, and safely ships to the cloud.
        </p>
        
        <div className="flex flex-col sm:flex-row gap-4 w-full sm:w-auto">
          <Button href="/docs" icon="rocket" className="shadow-lg shadow-blue-500/25 hover:shadow-blue-500/40 hover:-translate-y-0.5 transition-all duration-300">
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

      {/* Core Features Grid */}
      <section className="w-full max-w-7xl px-4 py-20 relative z-10">
        <div className="text-center mb-16">
          <h2 className="text-4xl font-bold tracking-tight mb-4">A Self-Healing Pipeline</h2>
          <p className="text-gray-500 dark:text-gray-400 text-lg max-w-2xl mx-auto">
            Not just another wrapper. Atlas enforces strict preconditions, orchestrates your build, and leverages bounded AI to safely fix errors.
          </p>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {/* Feature 1 */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-blue-100 dark:bg-blue-900/50 flex items-center justify-center text-blue-600 dark:text-blue-400 mb-6 group-hover:scale-110 transition-transform">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" /></svg>
            </div>
            <h3 className="text-xl font-bold mb-3">Zero Config Analysis</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              Atlas automatically detects your framework, package manager, and required databases to assemble the perfect build command without any configuration files.
            </p>
          </div>

          {/* Feature 2 */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-teal-100 dark:bg-teal-900/50 flex items-center justify-center text-teal-600 dark:text-teal-400 mb-6 group-hover:scale-110 transition-transform">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" /></svg>
            </div>
            <h3 className="text-xl font-bold mb-3">LLM Diagnostics</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              If a local build fails, Atlas captures the stack trace and uses models like Claude, OpenAI, or Gemini to diagnose the issue and propose a fix.
            </p>
          </div>

          {/* Feature 3 */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-indigo-100 dark:bg-indigo-900/50 flex items-center justify-center text-indigo-600 dark:text-indigo-400 mb-6 group-hover:scale-110 transition-transform">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>
            </div>
            <h3 className="text-xl font-bold mb-3">Exact-Match Patching</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              No free-form file rewrites. Atlas uses sandboxed, string-replacement logic for precise edits, rolling back via Git if the fix fails.
            </p>
          </div>

          {/* Feature 4 */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-orange-100 dark:bg-orange-900/50 flex items-center justify-center text-orange-600 dark:text-orange-400 mb-6 group-hover:scale-110 transition-transform">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" /></svg>
            </div>
            <h3 className="text-xl font-bold mb-3">Database Self-Healing</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              Automatically provisions free-tier Postgres instances for Django. If you delete your cloud resources, Atlas detects the 404 and safely re-provisions them.
            </p>
          </div>

          {/* Feature 5 */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-purple-100 dark:bg-purple-900/50 flex items-center justify-center text-purple-600 dark:text-purple-400 mb-6 group-hover:scale-110 transition-transform">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            </div>
            <h3 className="text-xl font-bold mb-3">Health Checks & Rollback</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              A successful build isn't enough. Atlas runs post-deployment health checks against your live URL. If it fails, you are prompted to rollback the deployment.
            </p>
          </div>

          {/* Feature 6 */}
          <div className="p-8 rounded-2xl border border-gray-200/80 dark:border-zinc-800/80 bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl hover:scale-[1.02] hover:shadow-2xl transition-all duration-300 group">
            <div className="w-12 h-12 rounded-lg bg-pink-100 dark:bg-pink-900/50 flex items-center justify-center text-pink-600 dark:text-pink-400 mb-6 group-hover:scale-110 transition-transform">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" /></svg>
            </div>
            <h3 className="text-xl font-bold mb-3">Secure Secrets</h3>
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
              Provider tokens and ephemeral keys (like Django's SECRET_KEY) are stored securely in your OS keychain, never saved to plaintext logs or files.
            </p>
          </div>
        </div>
      </section>

      {/* Tech Stack Matrix */}
      <section className="w-full bg-white dark:bg-zinc-900 border-y border-gray-200 dark:border-zinc-800 py-16 px-4">
        <div className="max-w-5xl mx-auto flex flex-col md:flex-row gap-12 justify-around items-start">
          
          <div className="flex-1">
            <h3 className="text-sm font-bold tracking-widest uppercase text-gray-400 mb-6">Supported Frameworks</h3>
            <ul className="space-y-4">
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600 text-xs">✓</span>
                Next.js / React
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600 text-xs">✓</span>
                Django (with DB auto-provisioning)
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600 text-xs">✓</span>
                Express / Node.js
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600 text-xs">✓</span>
                Go
              </li>
            </ul>
          </div>
          
          <div className="flex-1">
            <h3 className="text-sm font-bold tracking-widest uppercase text-gray-400 mb-6">Supported Providers</h3>
            <ul className="space-y-4">
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-blue-100 text-blue-600 text-xs">✓</span>
                Render (Web Services & PostgreSQL)
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-blue-100 text-blue-600 text-xs">✓</span>
                Vercel
              </li>
              <li className="flex items-center gap-3 text-lg font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-blue-100 text-blue-600 text-xs">✓</span>
                Netlify
              </li>
              <li className="flex items-center gap-3 text-lg font-medium text-gray-400">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-800 text-gray-500 text-xs">○</span>
                Fly.io & Railway (Coming soon)
              </li>
            </ul>
          </div>
          
        </div>
      </section>

      {/* FAQ Section */}
      <section className="w-full max-w-4xl px-4 py-24 mb-10">
        <h2 className="text-3xl font-bold mb-12 text-center">Frequently Asked Questions</h2>
        <div className="space-y-4">
          <Accordion title="Which LLM models are supported?" icon="cpu">
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed p-4">
              Atlas natively supports Anthropic (Claude), OpenAI, Google Gemini, Mistral, Groq, xAI (Grok), and even local models via Ollama. 
              The models are strictly used for the `FixCode` diagnostic phase, meaning you control when and how AI is invoked.
            </p>
          </Accordion>
          
          <Accordion title="Is it going to overwrite my code silently?" icon="shield">
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed p-4">
              <strong>Absolutely not.</strong> Atlas operates as a deterministic pipeline. It does not use full-file generation.
              When an LLM proposes a fix, it is strictly bound to an exact-match patch (`old_str` / `new_str`). Consequential actions 
              like pushing a fix to the remote repository or triggering a deployment require explicit manual confirmation.
            </p>
          </Accordion>
          
          <Accordion title="How are Django projects deployed?" icon="database">
            <p className="text-gray-600 dark:text-gray-400 leading-relaxed p-4">
              Atlas features a dedicated, rigorous pre-flight checklist for Django. It forces you to drop SQLite for production, checks 
              for env-controlled `SECRET_KEY` and `DEBUG`, verifies `WhiteNoiseMiddleware`, and auto-provisions a secure PostgreSQL 
              database on Render automatically.
            </p>
          </Accordion>
        </div>
      </section>
    </div>
  );
}
