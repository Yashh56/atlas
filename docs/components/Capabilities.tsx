const Card = ({ className, ...props }: any) => (
  <div className={`rounded-xl border border-zinc-200 bg-white text-zinc-950 shadow dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-50 ${className}`} {...props} />
);

const CardHeader = ({ className, ...props }: any) => (
  <div className={`flex flex-col space-y-1.5 p-6 ${className}`} {...props} />
);

const CardTitle = ({ className, ...props }: any) => (
  <h3 className={`font-semibold leading-none tracking-tight ${className}`} {...props} />
);

const CardContent = ({ className, ...props }: any) => (
  <div className={`p-6 pt-0 ${className}`} {...props} />
);

export function Capabilities() {
  return (
    <section className="w-full max-w-5xl mx-auto px-4 py-20 relative z-10 border-t border-zinc-200 dark:border-zinc-800">
      <div className="flex flex-col mb-12">
        <h2 className="text-3xl font-bold tracking-tight mb-4 text-zinc-950 dark:text-zinc-50">
          Built-in Capabilities
        </h2>
        <p className="text-zinc-600 dark:text-zinc-400 text-lg">
          Powerful features that work completely out of the box.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="flex flex-col justify-start">
          <CardHeader>
            <div className="w-10 h-10 rounded-lg bg-zinc-100 dark:bg-zinc-900 flex items-center justify-center mb-2 border border-zinc-200 dark:border-zinc-800">
              <svg className="w-5 h-5 text-zinc-900 dark:text-zinc-50" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" /></svg>
            </div>
            <CardTitle>Database Provisioning</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Automatically provisions free-tier Postgres instances for Django on Render, and re-provisions them if manually deleted.
            </p>
          </CardContent>
        </Card>

        <Card className="flex flex-col justify-start">
          <CardHeader>
            <div className="w-10 h-10 rounded-lg bg-zinc-100 dark:bg-zinc-900 flex items-center justify-center mb-2 border border-zinc-200 dark:border-zinc-800">
              <svg className="w-5 h-5 text-zinc-900 dark:text-zinc-50" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" /></svg>
            </div>
            <CardTitle>Secure Secrets</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Provider tokens and ephemeral keys (like Django's SECRET_KEY) are stored securely in your OS keychain, never saved to plaintext logs.
            </p>
          </CardContent>
        </Card>

        <Card className="flex flex-col justify-start">
          <CardHeader>
            <div className="w-10 h-10 rounded-lg bg-zinc-100 dark:bg-zinc-900 flex items-center justify-center mb-2 border border-zinc-200 dark:border-zinc-800">
              <svg className="w-5 h-5 text-zinc-900 dark:text-zinc-50" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg>
            </div>
            <CardTitle>Multi-LLM Support</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Choose from Anthropic, OpenAI, Gemini, Mistral, Groq, xAI, or run completely offline with Ollama for the FixCode diagnostic step.
            </p>
          </CardContent>
        </Card>
      </div>
    </section>
  );
}
