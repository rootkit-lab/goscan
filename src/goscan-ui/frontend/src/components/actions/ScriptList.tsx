import { FastForward, Layers, Play, RotateCcw, Square } from "lucide-react";
import { BatchProgressBar } from "@/components/batch/BatchProgressBar";
import { CheckerStatusIcon, statusRowClass } from "@/components/checkers/CheckerStatusIcon";
import type { BatchProgressDTO, ScriptCheckerStatusDTO } from "@/lib/api";
import { scriptIcon, statusTitle, type CheckerStatus } from "@/lib/scriptIcons";

type Props = {
  scripts: ScriptCheckerStatusDTO[];
  selectedScript: string;
  onScriptChange: (id: string) => void;
  onRunScript: (id?: string) => void;
  onCancelScript?: () => void;
  onTestAllFinding?: () => void;
  onTestAllFiltered?: () => void;
  onTestAllQuick?: () => void;
  onTestAllEnvs?: () => void;
  onCancelBatch?: () => void;
  batchRunning?: boolean;
  batchProgress?: BatchProgressDTO | null;
  batchLabel?: string;
  batchThreads: number;
  onBatchThreadsChange: (n: number) => void;
  terminalActive?: boolean;
  runningScriptId?: string;
};

export function ScriptList({
  scripts,
  selectedScript,
  onScriptChange,
  onRunScript,
  onCancelScript,
  onTestAllFinding,
  onTestAllFiltered,
  onTestAllQuick,
  onTestAllEnvs,
  onCancelBatch,
  batchRunning,
  batchProgress,
  batchLabel,
  batchThreads,
  onBatchThreadsChange,
  terminalActive,
  runningScriptId
}: Props) {
  return (
    <div className="space-y-1">
      {scripts.length === 0 ? (
        <p className="text-[11px] text-vscode-muted">Nenhum checker compatível neste finding</p>
      ) : (
        <>
      <div className="grid grid-cols-[1fr_auto_auto] gap-x-1 border-b border-vscode-border pb-1 text-[10px] uppercase tracking-wide text-vscode-muted">
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
              <Icon className="h-3.5 w-3.5 shrink-0 text-vscode-muted" />
              <span className="truncate text-vscode-fg">{s.label}</span>
            </button>
            <span className="flex w-8 justify-center">
              <CheckerStatusIcon status={status} />
            </span>
            <span className="flex w-14 justify-center gap-0.5">
              <button
                type="button"
                className="p-0.5 hover:bg-vscode-selection disabled:opacity-40"
                disabled={status === "running"}
                onClick={() => onRunScript(s.scriptId)}
                title="Executar"
              >
                <Play className="h-3 w-3" />
              </button>
              {status !== "pending" && status !== "running" && (
                <button
                  type="button"
                  className="p-0.5 hover:bg-vscode-selection"
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
        </>
      )}

      <BatchProgressBar progress={batchProgress ?? null} running={!!batchRunning} />

      <div className="flex flex-col gap-1 border-t border-vscode-border pt-2">
        <p className="text-[10px] uppercase tracking-wide text-vscode-muted">Batch envs</p>
        <label className="flex items-center gap-2 text-[11px] text-vscode-fg">
          <span className="shrink-0 text-vscode-muted">Threads</span>
          <input
            type="number"
            min={1}
            max={16}
            className="vscode-input w-16 py-0.5 text-[11px]"
            value={batchThreads}
            disabled={batchRunning}
            onChange={(e) => onBatchThreadsChange(Math.max(1, Math.min(16, Number(e.target.value) || 1)))}
          />
        </label>
        <button
          type="button"
          className="vscode-btn vscode-btn-primary flex w-full items-center justify-center gap-1 text-[11px]"
          disabled={batchRunning || !!runningScriptId}
          onClick={() => onTestAllEnvs?.()}
          title="Testa todos os .env do filtro actual em paralelo"
        >
          <Layers className="h-3.5 w-3.5" />
          Test all envs
        </button>
        <div className="flex gap-1">
          <button
            type="button"
            className="vscode-btn vscode-btn-primary flex flex-1 items-center justify-center gap-1 text-[11px]"
            disabled={!selectedScript || !!runningScriptId || batchRunning}
            onClick={() => onRunScript()}
          >
            <Play className="h-3.5 w-3.5" />
            Run selected
          </button>
          {terminalActive && onCancelScript && (
            <button type="button" className="vscode-btn px-2" onClick={onCancelScript} title="Parar">
              <Square className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
        <button
          type="button"
          className="vscode-btn flex w-full items-center justify-center gap-1 text-[11px]"
          disabled={scripts.length === 0 || !!runningScriptId || batchRunning}
          onClick={() => onTestAllFinding?.()}
        >
          <FastForward className="h-3.5 w-3.5" />
          Test all (finding)
        </button>
        <button
          type="button"
          className="vscode-btn flex w-full items-center justify-center gap-1 text-[11px]"
          disabled={batchRunning}
          onClick={() => onTestAllFiltered?.()}
        >
          <FastForward className="h-3.5 w-3.5" />
          Test all (filtro)
        </button>
        <button
          type="button"
          className="vscode-btn flex w-full items-center justify-center gap-1 text-[11px]"
          disabled={batchRunning}
          onClick={() => onTestAllQuick?.()}
          title="Sem email nem DB pesado"
        >
          Test all quick
        </button>
        {batchRunning && onCancelBatch && (
          <button type="button" className="vscode-btn flex w-full items-center justify-center gap-1 text-[11px]" onClick={onCancelBatch}>
            <Square className="h-3.5 w-3.5" />
            Parar batch
          </button>
        )}
        {batchLabel && !batchRunning && <p className="truncate text-[10px] text-vscode-muted">{batchLabel}</p>}
      </div>
    </div>
  );
}
