import { Play, RotateCcw, Square } from "lucide-react";
import { CheckerStatusIcon, statusRowClass } from "@/components/checkers/CheckerStatusIcon";
import type { ScriptCheckerStatusDTO } from "@/lib/api";
import { scriptIcon, statusTitle, type CheckerStatus } from "@/lib/scriptIcons";

type Props = {
  scripts: ScriptCheckerStatusDTO[];
  selectedScript: string;
  onScriptChange: (id: string) => void;
  onRunScript: (id?: string) => void;
  onCancelScript?: () => void;
  terminalActive?: boolean;
  runningScriptId?: string;
};

export function ScriptList({
  scripts,
  selectedScript,
  onScriptChange,
  onRunScript,
  onCancelScript,
  terminalActive,
  runningScriptId
}: Props) {
  if (scripts.length === 0) {
    return <p className="text-[11px] text-gs-muted">Nenhum checker compatível neste finding</p>;
  }

  return (
    <div className="space-y-1">
      <div className="grid grid-cols-[1fr_auto_auto] gap-x-1 border-b border-gs-border pb-1 text-[10px] uppercase tracking-wide text-gs-muted">
        <span>Checker</span>
        <span className="w-8 text-center">Est.</span>
        <span className="w-14 text-center">Acção</span>
      </div>
      {scripts.map((s) => {
        const status: CheckerStatus = runningScriptId === s.scriptId ? "running" : (s.status as CheckerStatus);
        const Icon = scriptIcon(s.scriptId);
        const selected = selectedScript === s.scriptId;
        return (
          <div
            key={s.scriptId}
            className={statusRowClass(status, selected)}
            title={statusTitle(status, s.summary) + (s.logPath ? `\nLog: ${s.logPath}` : "")}
          >
            <button type="button" className="flex min-w-0 flex-1 items-center gap-1.5" onClick={() => onScriptChange(s.scriptId)}>
              <Icon className="h-3.5 w-3.5 shrink-0 text-gs-muted" />
              <span className="truncate text-gs-fg">{s.label}</span>
            </button>
            <span className="flex w-8 justify-center">
              <CheckerStatusIcon status={status} />
            </span>
            <span className="flex w-14 justify-center gap-0.5">
              <button
                type="button"
                className="p-0.5 hover:bg-gs-selection disabled:opacity-40"
                disabled={status === "running"}
                onClick={() => onRunScript(s.scriptId)}
                title="Executar"
              >
                <Play className="h-3 w-3" />
              </button>
              {status !== "pending" && status !== "running" && (
                <button
                  type="button"
                  className="p-0.5 hover:bg-gs-selection"
                  onClick={() => onRunScript(s.scriptId)}
                  title="Re-testar"
                >
                  <RotateCcw className="h-3 w-3" />
                </button>
              )}
            </span>
          </div>
        );
      })}

      <div className="flex gap-1 border-t border-gs-border pt-2">
        <button
          type="button"
          className="gs-btn gs-btn-primary flex flex-1 items-center justify-center gap-1 text-[11px]"
          disabled={!selectedScript || !!runningScriptId}
          onClick={() => onRunScript()}
        >
          <Play className="h-3.5 w-3.5" />
          Run selected
        </button>
        {terminalActive && onCancelScript && (
          <button type="button" className="gs-btn px-2" onClick={onCancelScript} title="Parar">
            <Square className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
    </div>
  );
}
