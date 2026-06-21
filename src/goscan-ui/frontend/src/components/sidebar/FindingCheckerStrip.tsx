import { CheckerStatusIcon } from "@/components/checkers/CheckerStatusIcon";
import type { ScriptCheckerStatusDTO } from "@/lib/api";
import { type CheckerResultFilter, visibleCheckerScripts } from "@/lib/checkerFilters";
import { scriptIcon, scriptShortLabel, statusTitle } from "@/lib/scriptIcons";
import { clsx } from "clsx";

type Props = {
  scripts: ScriptCheckerStatusDTO[];
  runningScriptId?: string;
  checkerFilter?: CheckerResultFilter;
  maxVisible?: number;
  compact?: boolean;
};

export function FindingCheckerStrip({
  scripts,
  runningScriptId,
  checkerFilter = "",
  maxVisible = 6,
  compact = false
}: Props) {
  const shown = visibleCheckerScripts(scripts, checkerFilter);
  if (shown.length === 0) return null;

  const visible = shown.slice(0, maxVisible);
  const extra = shown.length - visible.length;

  return (
    <div className={clsx("flex flex-wrap items-center gap-1", !compact && "mt-2")}>
      {visible.map((s) => {
        const status = runningScriptId === s.scriptId ? "running" : s.status;
        const Icon = scriptIcon(s.scriptId);
        const ok = status === "ok";
        return (
          <span
            key={s.scriptId}
            className={clsx(
              "inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5",
              ok
                ? "border-gs-success/25 bg-gs-success/10"
                : status === "fail"
                  ? "border-gs-error/25 bg-gs-error/10"
                  : "border-gs-border/60 bg-gs-surface-raised/80"
            )}
            title={`${s.label} — ${statusTitle(status as "ok", s.summary)}`}
          >
            <Icon className="h-2.5 w-2.5 text-gs-muted" />
            <span className="text-[10px] text-gs-muted">{scriptShortLabel(s.label)}</span>
            <CheckerStatusIcon status={status as "ok"} size="sm" />
          </span>
        );
      })}
      {extra > 0 && <span className="px-1 text-[10px] text-gs-muted">+{extra}</span>}
    </div>
  );
}
