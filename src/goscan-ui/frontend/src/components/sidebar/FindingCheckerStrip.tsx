import { CheckerStatusIcon } from "@/components/checkers/CheckerStatusIcon";
import type { ScriptCheckerStatusDTO } from "@/lib/api";
import { type CheckerResultFilter, visibleCheckerScripts } from "@/lib/checkerFilters";
import { scriptIcon, scriptShortLabel, statusTitle } from "@/lib/scriptIcons";

type Props = {
  scripts: ScriptCheckerStatusDTO[];
  runningScriptId?: string;
  checkerFilter?: CheckerResultFilter;
  maxVisible?: number;
};

export function FindingCheckerStrip({
  scripts,
  runningScriptId,
  checkerFilter = "",
  maxVisible = 6
}: Props) {
  const shown = visibleCheckerScripts(scripts, checkerFilter);
  if (shown.length === 0) return null;

  const visible = shown.slice(0, maxVisible);
  const extra = shown.length - visible.length;

  return (
    <div className="mt-0.5 flex flex-wrap items-center gap-1">
      {visible.map((s) => {
        const status = runningScriptId === s.scriptId ? "running" : s.status;
        const Icon = scriptIcon(s.scriptId);
        return (
          <span
            key={s.scriptId}
            className="inline-flex items-center gap-0.5 rounded-sm bg-vscode-input/40 px-1 py-px"
            title={`${s.label} — ${statusTitle(status as "ok", s.summary)}`}
          >
            <Icon className="h-2.5 w-2.5 text-vscode-muted" />
            <span className="text-[9px] text-vscode-muted">{scriptShortLabel(s.label)}</span>
            <CheckerStatusIcon status={status as "ok"} size="sm" />
          </span>
        );
      })}
      {extra > 0 && <span className="text-[9px] text-vscode-muted">+{extra}</span>}
    </div>
  );
}
