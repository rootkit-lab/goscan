import { Copy, FolderOpen, Trash2 } from "lucide-react";
import type { RefObject } from "react";
import { useEffect, useRef } from "react";
import { clsx } from "clsx";
import { CheckerStatusIcon } from "@/components/checkers/CheckerStatusIcon";
import { ResizablePanel } from "@/components/layout/ResizablePanel";
import type { ScriptCheckerStatusDTO } from "@/lib/api";
import { scriptIcon, statusTitle, type CheckerStatus } from "@/lib/scriptIcons";

export type BottomTab = "output" | "terminal" | "results" | "batch-log";

export const BATCH_BOTTOM_TABS: BottomTab[] = ["batch-log"];
export const EDITOR_BOTTOM_TABS: BottomTab[] = ["terminal", "results"];

type Props = {
  tabs?: BottomTab[];
  tab: BottomTab;
  onTabChange: (tab: BottomTab) => void;
  outputLines: string[];
  batchLogLines: string[];
  onClearOutput: () => void;
  onClearBatchLog: () => void;
  logLines?: string[];
  onClearLog?: () => void;
  termRef?: RefObject<HTMLDivElement | null>;
  onClearTerminal?: () => void;
  terminalActive?: boolean;
  batchLogDir?: string;
  onOpenBatchLogs?: () => void;
  resultsScripts?: ScriptCheckerStatusDTO[];
  resultsFindingLabel?: string;
  runningScriptId?: string;
  defaultHeight?: number;
};

export function BottomPanel({
  tabs = ["output", "terminal", "results", "batch-log"],
  tab,
  onTabChange,
  outputLines,
  batchLogLines,
  onClearOutput,
  onClearBatchLog,
  logLines,
  onClearLog,
  termRef,
  onClearTerminal,
  terminalActive = false,
  batchLogDir,
  onOpenBatchLogs,
  resultsScripts = [],
  resultsFindingLabel,
  runningScriptId,
  defaultHeight = 200
}: Props) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const logScrollRef = useRef<HTMLDivElement>(null);
  const batchScrollRef = useRef<HTMLDivElement>(null);
  const hasLog = tabs.includes("terminal") && logLines !== undefined;
  const hasTerminal = tabs.includes("terminal") && !!termRef && !hasLog;

  useEffect(() => {
    if (tab === "output") {
      const el = scrollRef.current;
      if (el) el.scrollTop = el.scrollHeight;
    }
    if (tab === "batch-log") {
      const el = batchScrollRef.current;
      if (el) el.scrollTop = el.scrollHeight;
    }
    if (tab === "terminal" && hasLog) {
      const el = logScrollRef.current;
      if (el) el.scrollTop = el.scrollHeight;
    }
  }, [outputLines, batchLogLines, logLines, tab, hasLog]);

  const copyContent = () => {
    if (tab === "output") {
      void navigator.clipboard.writeText(outputLines.join("\n"));
      return;
    }
    if (tab === "batch-log") {
      void navigator.clipboard.writeText(batchLogLines.join("\n"));
      return;
    }
    if (tab === "results") {
      const text = resultsScripts.map((s) => `${s.label}\t${s.status}\t${s.summary ?? ""}`).join("\n");
      void navigator.clipboard.writeText(text);
      return;
    }
    if (tab === "terminal" && hasLog) {
      void navigator.clipboard.writeText(logLines.join("\n"));
      return;
    }
    if (hasTerminal && termRef?.current) {
      void navigator.clipboard.writeText(termRef.current.innerText);
    }
  };

  const clearContent = () => {
    if (tab === "output") onClearOutput();
    else if (tab === "batch-log") onClearBatchLog();
    else if (tab === "terminal" && hasLog) onClearLog?.();
    else if (tab === "terminal") onClearTerminal?.();
  };

  const canClear = tab !== "results";

  return (
    <ResizablePanel
      title=""
      defaultHeight={defaultHeight}
      minHeight={100}
      maxHeight={520}
      headerRight={
        <span className="flex w-full items-center justify-between gap-2">
          <TabBar tab={tab} tabs={tabs} onTabChange={onTabChange} terminalActive={terminalActive} />
          <span className="flex shrink-0 gap-1">
            {tab === "batch-log" && batchLogDir && onOpenBatchLogs ? (
              <button
                type="button"
                className="flex items-center gap-1 px-1 text-[10px] text-gs-muted hover:bg-gs-hover hover:text-gs-fg"
                onClick={onOpenBatchLogs}
                title={batchLogDir}
              >
                <FolderOpen className="h-3 w-3" />
                Logs
              </button>
            ) : null}
            {canClear && (
              <button type="button" className="p-0.5 hover:bg-gs-hover" onClick={clearContent} title="Limpar">
                <Trash2 className="h-3 w-3" />
              </button>
            )}
            <button type="button" className="p-0.5 hover:bg-gs-hover" onClick={copyContent} title="Copiar">
              <Copy className="h-3 w-3" />
            </button>
          </span>
        </span>
      }
    >
      <div className="relative h-full min-h-0">
        {tabs.includes("output") && (
          <div
            ref={scrollRef}
            className={clsx(
              "absolute inset-0 overflow-auto font-mono text-[12px] leading-relaxed text-gs-fg",
              tab !== "output" && "hidden"
            )}
          >
            {outputLines.length === 0 ? (
              <span className="text-gs-muted">Output do scan e mensagens gerais aparecem aqui ao iniciar um scan.</span>
            ) : (
              outputLines.map((line, i) => (
                <div key={i} className="whitespace-pre-wrap break-all">
                  {line}
                </div>
              ))
            )}
          </div>
        )}

        {hasLog && (
          <div
            ref={logScrollRef}
            className={clsx(
              "absolute inset-0 overflow-auto p-2 font-mono text-[12px] leading-relaxed text-gs-fg",
              tab !== "terminal" && "hidden"
            )}
          >
            {logLines.length === 0 ? (
              <span className="text-gs-muted">Output do checker aparece aqui ao correr Run selected.</span>
            ) : (
              logLines.map((line, i) => (
                <div key={i} className="whitespace-pre-wrap break-all">
                  {line}
                </div>
              ))
            )}
          </div>
        )}

        {hasTerminal && (
          <div
            ref={termRef as React.RefObject<HTMLDivElement>}
            className={clsx("absolute inset-0 w-full", tab !== "terminal" && "hidden")}
          />
        )}

        {tabs.includes("batch-log") && (
          <div
            ref={batchScrollRef}
            className={clsx(
              "absolute inset-0 overflow-auto font-mono text-[12px] leading-relaxed text-gs-fg",
              tab !== "batch-log" && "hidden"
            )}
          >
            {batchLogLines.length === 0 ? (
              <span className="text-gs-muted">Linhas de batch aparecem aqui durante operações em lote.</span>
            ) : (
              batchLogLines.map((line, i) => (
                <div key={i} className="whitespace-pre-wrap break-all">
                  {line}
                </div>
              ))
            )}
          </div>
        )}

        {tabs.includes("results") && (
          <div
            className={clsx(
              "absolute inset-0 overflow-auto p-2",
              tab !== "results" && "hidden"
            )}
          >
            {!resultsFindingLabel ? (
              <span className="text-[12px] text-gs-muted">Seleccione um finding para ver resultados dos checkers.</span>
            ) : resultsScripts.length === 0 ? (
              <span className="text-[12px] text-gs-muted">Nenhum checker compatível com {resultsFindingLabel}.</span>
            ) : (
              <>
                <p className="mb-2 text-[11px] text-gs-muted">{resultsFindingLabel}</p>
                <table className="w-full text-left text-[12px]">
                  <thead>
                    <tr className="border-b border-gs-border text-[10px] uppercase tracking-wide text-gs-muted">
                      <th className="pb-1 pr-2">Checker</th>
                      <th className="pb-1 pr-2">Estado</th>
                      <th className="pb-1">Resumo</th>
                    </tr>
                  </thead>
                  <tbody>
                    {resultsScripts.map((s) => {
                      const status: CheckerStatus =
                        runningScriptId === s.scriptId ? "running" : (s.status as CheckerStatus);
                      const Icon = scriptIcon(s.scriptId);
                      return (
                        <tr key={s.scriptId} className="border-b border-gs-border/50">
                          <td className="py-1.5 pr-2">
                            <span className="flex items-center gap-1.5" title={statusTitle(status, s.summary)}>
                              <Icon className="h-3.5 w-3.5 shrink-0 text-gs-muted" />
                              {s.label}
                            </span>
                          </td>
                          <td className="py-1.5 pr-2">
                            <CheckerStatusIcon status={status} />
                          </td>
                          <td className="py-1.5 text-gs-muted">{s.summary || "—"}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </>
            )}
          </div>
        )}
      </div>
    </ResizablePanel>
  );
}

const TAB_LABELS: Record<BottomTab, string> = {
  output: "Output",
  terminal: "Log",
  results: "Resultados",
  "batch-log": "Batch log"
};

function TabBar({
  tab,
  tabs,
  onTabChange,
  terminalActive
}: {
  tab: BottomTab;
  tabs: BottomTab[];
  onTabChange: (t: BottomTab) => void;
  terminalActive: boolean;
}) {
  return (
    <span className="flex items-end gap-0 normal-case tracking-normal">
      {tabs.map((t) => (
        <TabButton
          key={t}
          active={tab === t}
          onClick={() => onTabChange(t)}
          pulse={t === "terminal" && terminalActive && tab !== "terminal"}
        >
          {TAB_LABELS[t]}
        </TabButton>
      ))}
    </span>
  );
}

function TabButton({
  active,
  onClick,
  children,
  pulse
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
  pulse?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={clsx(
        "relative px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide",
        active ? "text-gs-accent" : "text-gs-muted hover:text-gs-fg"
      )}
    >
      {children}
      {pulse && <span className="absolute -right-0.5 top-0 h-1.5 w-1.5 rounded-full bg-gs-accent" />}
    </button>
  );
}
