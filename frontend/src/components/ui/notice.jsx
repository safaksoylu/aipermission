import { cn } from "../../lib/utils";

export function Notice({ tone = "neutral", className, children }) {
  const tones = {
    neutral: "border-stone-200 bg-stone-50 text-stone-700 dark-notice-neutral",
    good: "border-emerald-200 bg-emerald-50 text-emerald-800 dark-notice-good",
    warn: "border-amber-200 bg-amber-50 text-amber-900 dark-notice-warn",
    bad: "border-red-200 bg-red-50 text-red-800 dark-notice-bad",
  };
  return <div className={cn("rounded-md border p-3 text-sm", tones[tone], className)}>{children}</div>;
}
