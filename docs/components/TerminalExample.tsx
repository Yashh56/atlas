export function TerminalExample() {
  return (
    <section className="w-full max-w-3xl px-4 py-8 relative z-10 mx-auto mt-4">
      <div className="rounded-xl overflow-hidden border border-zinc-200 dark:border-zinc-800 bg-zinc-950 shadow-2xl">
        <div className="flex items-center px-4 py-3 border-b border-zinc-800 bg-zinc-900/50">
          <div className="flex space-x-2">
            <div className="w-3 h-3 rounded-full bg-red-500/90"></div>
            <div className="w-3 h-3 rounded-full bg-yellow-500/90"></div>
            <div className="w-3 h-3 rounded-full bg-green-500/90"></div>
          </div>
          <div className="mx-auto text-xs text-zinc-400 font-medium font-mono">bash</div>
        </div>
        <div className="p-6 text-sm font-mono text-zinc-300 overflow-x-auto text-left leading-relaxed">
          <div className="flex gap-2"><span className="text-pink-500">~</span><span className="text-zinc-100">$</span><span>atlas ./my-app</span></div>
          <div className="mt-4 flex items-center gap-2 text-zinc-400"><span className="text-green-400">✓</span> Detected framework: Next.js</div>
          <div className="flex items-center gap-2 text-zinc-400"><span className="text-green-400">✓</span> Build command: npm run build</div>
          <div className="flex items-center gap-2 text-zinc-400"><span className="text-green-400">✓</span> Build succeeded</div>
          <div className="flex items-center gap-2 text-zinc-400"><span className="text-green-400">✓</span> Deployed to Vercel</div>
          <div className="flex items-center gap-2 text-zinc-400"><span className="text-green-400">✓</span> Health check passed</div>
          <div className="mt-4 text-zinc-100 font-semibold">🚀 Live at: <a href="https://my-app.vercel.app" className="text-blue-400 hover:underline">https://my-app.vercel.app</a></div>
        </div>
      </div>
    </section>
  );
}
