export function TechStack() {
  return (
    <section className="w-full max-w-5xl mx-auto px-4 py-20 border-t border-zinc-200 dark:border-zinc-800">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-12">
        <div>
          <h3 className="text-lg font-semibold text-zinc-950 dark:text-zinc-50 mb-6 flex items-center">
            <svg className="w-5 h-5 mr-2 text-zinc-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" /></svg>
            Supported Frameworks
          </h3>
          <ul className="space-y-4">
            {['Next.js / React', 'Express / Node.js', 'Django (Render and Vercel only)', 'Go'].map((item) => (
              <li key={item} className="flex items-center text-zinc-700 dark:text-zinc-300">
                <svg className="w-5 h-5 mr-3 text-zinc-900 dark:text-zinc-50" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                {item}
              </li>
            ))}
          </ul>
        </div>

        <div>
          <h3 className="text-lg font-semibold text-zinc-950 dark:text-zinc-50 mb-6 flex items-center">
            <svg className="w-5 h-5 mr-2 text-zinc-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" /></svg>
            Supported Providers
          </h3>
          <ul className="space-y-4">
            {['Render (Web Services & PostgreSQL)', 'Vercel', 'Netlify'].map((item) => (
              <li key={item} className="flex items-center text-zinc-700 dark:text-zinc-300">
                <svg className="w-5 h-5 mr-3 text-zinc-900 dark:text-zinc-50" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                {item}
              </li>
            ))}
            <li className="flex items-center text-zinc-500 dark:text-zinc-500">
              <svg className="w-5 h-5 mr-3 text-zinc-300 dark:text-zinc-700" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
              Fly.io & Railway (Coming soon)
            </li>
          </ul>
        </div>
      </div>
    </section>
  );
}
