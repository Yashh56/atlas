import { useEffect } from "react";
import { Hero } from "../../components/Hero"
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
      {/* Background ambient glow */}
      <div className="absolute top-0 inset-x-0 h-[500px] bg-gradient-to-b from-zinc-200/50 to-transparent pointer-events-none blur-3xl opacity-50 dark:from-zinc-800/20 dark:opacity-30"></div>
      <Hero />
      <TerminalExample />
      <SafetyCallout />
      <PipelineSteps />
      <Capabilities />
      <TechStack />
      <FAQ />
    </div>
  );
}
