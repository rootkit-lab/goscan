import { Copy, FolderOpen, Trash2 } from "lucide-react";
import type { RefObject } from "react";
import { useEffect, useRef } from "react";
import { clsx } from "clsx";
import { BatchProgressBar } from "@/components/batch/BatchProgressBar";
import { ResizablePanel } from "@/components/layout/ResizablePanel";
import type { BatchProgressDTO } from "@/lib/api";

export type BottomTab = "output" | "terminal";

type Props = {
  tab: BottomTab;
  onTabChange: (tab: BottomTab) => void;
  outputLines: string[];
  onClearOutput: () => void;
  termRef: RefObject<HTMLDivElement | null>;
  onClearTerminal: () => void;
  terminalActive: boolean;
  batchRunning?: boolean;
  batchProgress?: BatchProgressDTO | null;
  batchLogDir?: string;
  onOpenBatchLogs?: () => void;
};

export function BottomPanel({
  tab,
  onTabChange,
  outputLines,
  onClearOutput,
  termRef,
  onClearTerminal,
  terminalActive,
  batchRunning,
  batchProgress,
  batchLogDir,
  onOpenBatchLogs
}: Props) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (tab !== "output") return;
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [outputLines, tab]);

  const copyContent = () => {
    if (tab === "output") {
      void navigator.clipboard.writeText(outputLines.join("\n"));
      return;
    }
    const el = termRef.current;
    if (el) void navigator.clipboard.writeText(el.innerText);
  };

  const clearContent = () => {
    if (tab === "output") onClearOutput();
    else onClearTerminal();
  };

  return (
    <ResizablePanel
      title=""
      headerRight={
        <span className="flex w-full items-center justify-between gap-2">
          <TabBar tab={tab} onTabChange={onTabChange} terminalActive={terminalActive} />
          <span className="flex shrink-0 gap-1">
            {batchLogDir && onOpenBatchLogs ? (
              <button
                type="button"
                className="flex items-center gap-1 px-1 text-[10px] text-vscode-muted hover:bg-vscode-hover hover:text-vscode-fg"
                onClick={onOpenBatchLogs}
                title={batchLogDir}
              >
                <FolderOpen className="h-3 w-3" />
                Logs
              </button>
            ) : null}
            <button type="button" className="p-0.5 hover:bg-vscode-hover" onClick={clearContent} title="Limpar">
              <Trash2 className="h-3 w-3" />
            </button>
            <button type="button" className="p-0.5 hover:bg-vscode-hover" onClick={copyContent} title="Copiar">
              <Copy className="h-3 w-3" />
            </button>
          </span>
        </span>
      }
    >
      <div className="relative flex h-full flex-col">
        <BatchProgressBar progress={batchProgress ?? null} running={!!batchRunning && tab === "output"} />
        <div
          ref={scrollRef}
          className={clsx(
            "min-h-0 flex-1 overflow-auto font-mono text-[12px] leading-relaxed text-vscode-fg",
            tab !== "output" && "hidden"
          )}
        >
          {outputLines.length === 0 ? (
            <span className="text-vscode-muted">Resultados do scan aparecem aqui.</span>
          ) : (
            outputLines.map((line, i) => (
              <div key={i} className="whitespace-pre-wrap break-all">
                {line}
              </div>
            ))
          )}
        </div>
        <div
          ref={termRef as React.RefObject<HTMLDivElement>}
          className={clsx("min-h-0 flex-1 w-full", tab !== "terminal" && "hidden")}
        />
      </div>
    </ResizablePanel>
  );
}

function TabBar({
  tab,
  onTabChange,
  terminalActive
}: {
  tab: BottomTab;
  onTabChange: (t: BottomTab) => void;
  terminalActive: boolean;
}) {
  return (
    <span className="flex items-end gap-0 normal-case tracking-normal">
      <TabButton active={tab === "output"} onClick={() => onTabChange("output")}>
        Output
      </TabButton>
      <TabButton active={tab === "terminal"} onClick={() => onTabChange("terminal")} pulse={terminalActive && tab !== "terminal"}>
        Terminal
      </TabButton>
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
        active ? "text-vscode-accent" : "text-vscode-muted hover:text-vscode-fg"
      )}
    >
      {children}
      {pulse && <span className="absolute -right-0.5 top-0 h-1.5 w-1.5 rounded-full bg-vscode-accent" />}
    </button>
  );
}
