import { Check, Copy } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "./button";

export function CopyButton({ value, children = "Copy", className = "", iconClassName = "h-4 w-4", disabled, onCopied, ...props }) {
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState(false);
  const timerRef = useRef(null);

  useEffect(() => {
    return () => window.clearTimeout(timerRef.current);
  }, []);

  async function copyValue(event) {
    props.onClick?.(event);
    if (event.defaultPrevented || disabled) return;
    try {
      await navigator.clipboard.writeText(String(value ?? ""));
      setCopied(true);
      setCopyError(false);
      onCopied?.();
    } catch {
      setCopied(false);
      setCopyError(true);
    }
    window.clearTimeout(timerRef.current);
    timerRef.current = window.setTimeout(() => {
      setCopied(false);
      setCopyError(false);
    }, 1200);
  }

  const Icon = copied ? Check : Copy;

  return (
    <Button
      {...props}
      type={props.type || "button"}
      className={className}
      disabled={disabled}
      title={copyError ? "Copy failed" : copied ? "Copied" : props.title}
      onClick={copyValue}
    >
      <Icon className={`${iconClassName} ${copied ? "text-emerald-700" : copyError ? "text-red-700" : ""}`} />
      {children}
    </Button>
  );
}
