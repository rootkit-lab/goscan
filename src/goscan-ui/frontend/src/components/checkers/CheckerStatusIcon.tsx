import { clsx } from "clsx";
import { Check, Circle, Loader2, Minus, X } from "lucide-react";
import type { CheckerStatus } from "@/lib/scriptIcons";

type Props = {
  status: CheckerStatus;
  className?: string;
  size?: "sm" | "md";
};

export function CheckerStatusIcon({ status, className, size = "sm" }: Props) {
  const sz = size === "sm" ? "h-3 w-3" : "h-3.5 w-3.5";
  const base = clsx("shrink-0", sz, className);

  switch (status) {
    case "ok":
      return <Check className={clsx(base, "text-[#89d185]")} strokeWidth={2.5} />;
    case "fail":
      return <X className={clsx(base, "text-vscode-error")} strokeWidth={2.5} />;
    case "skip":
      return <Minus className={clsx(base, "text-vscode-warning")} strokeWidth={2.5} />;
    case "running":
      return <Loader2 className={clsx(base, "animate-spin text-vscode-accent")} strokeWidth={2.5} />;
    default:
      return <Circle className={clsx(base, "text-vscode-dim")} strokeWidth={1.5} />;
  }
}

export function statusRowClass(status: CheckerStatus, selected?: boolean): string {
  return clsx(
    "flex w-full items-center gap-2 px-2 py-1.5 text-left text-[12px] hover:bg-vscode-hover",
    selected && "bg-vscode-list-active"
  );
}
