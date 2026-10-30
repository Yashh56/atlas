import { useEffect } from "react";
import { Hero } from "../../components/Hero";
import { TerminalExample } from "../../components/TerminalExample";
import { SafetyCallout } from "../../components/SafetyCallout";
import { PipelineSteps } from "../../components/PipelineSteps";
import { Capabilities } from "../../components/Capabilities";
import { TechStack } from "../../components/TechStack";
import { FAQ } from "../../components/FAQ";

export const frontmatter = {
  title: "Atlas",
  description: "The autonomous, self-healing deployment pipeline.",
  search: false,
};

export default function Home() {
  useEffect(() => {
    // Inject a temporary style tag to hide the shiso footer attribution only on this page
    const style = document.createElement("style");
    style.innerHTML = `a[href="https://shiso.umami.is?ref=docs-footer"] { display: none !important; }`;
    document.head.appendChild(style);
    return () => {
      document.head.removeChild(style);
    };
  }, []);

  return (
    <div className="flex flex-col items-center min-h-screen bg-white dark:bg-zinc-950 overflow-hidden relative font-sans selection:bg-zinc-900 selection:text-white dark:selection:bg-zinc-100 dark:selection:text-zinc-900">
      {/* Background ambient radial and gradient glow */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[1100px] h-[600px] bg-gradient-to-b from-emerald-500/10 via-zinc-200/40 to-transparent pointer-events-none blur-3xl opacity-70 dark:from-emerald-500/5 dark:via-zinc-800/25 dark:opacity-40" />
      <div className="absolute top-[800px] left-1/2 -translate-x-1/2 w-[900px] h-[500px] bg-gradient-to-b from-blue-500/5 to-transparent pointer-events-none blur-3xl opacity-50 dark:from-blue-500/5 dark:opacity-30" />

      {/* Subtle background grid pattern */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#8080800d_1px,transparent_1px),linear-gradient(to_bottom,#8080800d_1px,transparent_1px)] bg-[size:32px_32px] pointer-events-none" />

      {/* Main content sections with unified container rhythm */}
      <div className="w-full relative z-10 flex flex-col items-center">
        <Hero />
        <TerminalExample />
        <SafetyCallout />
        <PipelineSteps />
        <Capabilities />
        <TechStack />
        <FAQ />
      </div>
    </div>
  );
}
