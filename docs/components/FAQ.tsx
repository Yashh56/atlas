import { useState } from "react";

function FAQItem({ title, children }: { title: string, children: React.ReactNode }) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="border-b border-zinc-200 dark:border-zinc-800">
      <button
        type="button"
        className="flex flex-1 items-center justify-between py-4 font-medium transition-all hover:underline text-left w-full text-zinc-950 dark:text-zinc-50"
        onClick={() => setIsOpen(!isOpen)}
      >
        {title}
        <svg
          className={`h-4 w-4 shrink-0 transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`}
          fill="none" viewBox="0 0 24 24" stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {isOpen && (
        <div className="pb-4 pt-0 text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed">
          {children}
        </div>
      )}
    </div>
  );
}

export function FAQ() {
  return (
    <section className="w-full max-w-3xl mx-auto px-4 py-20 border-t border-zinc-200 dark:border-zinc-800 mb-10">
      <div className="flex flex-col mb-10 text-center">
        <h2 className="text-3xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50">
          Frequently Asked Questions
        </h2>
      </div>
      <div className="space-y-1">
        <FAQItem title="Which LLM models are supported?">
          Atlas natively supports Anthropic (Claude), OpenAI, Google Gemini, Mistral, Groq, xAI (Grok), and even local models via Ollama. The models are strictly used for the <code>FixCode</code> diagnostic phase, meaning you control when and how AI is invoked.
        </FAQItem>

        <FAQItem title="How are Django projects deployed?">
          Atlas features a dedicated, rigorous pre-flight checklist for Django. It forces you to drop SQLite for production, checks for env-controlled <code>SECRET_KEY</code> and <code>DEBUG</code>, verifies <code>WhiteNoiseMiddleware</code>, and auto-provisions a secure PostgreSQL database on Render automatically. Django deployment is currently supported on Render only.
        </FAQItem>

        <FAQItem title="Can I use Atlas in CI/CD pipelines?">
          Yes. Atlas supports fully non-interactive mode via flags: <code>atlas . --action deploy --provider vercel</code>. Provide your LLM and provider API keys as environment variables, and Atlas will run without any interactive prompts.
        </FAQItem>
      </div>
    </section>
  );
}
