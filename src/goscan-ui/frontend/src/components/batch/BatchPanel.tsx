import { FastForward, Filter, FolderOpen, Layers, Zap } from "lucide-react";
import { clsx } from "clsx";
import { useEffect, useState } from "react";
import { BatchLogView } from "@/components/batch/BatchLogView";
import { BatchProgressBar } from "@/components/batch/BatchProgressBar";
import type { BatchResultRow } from "@/components/batch/batchResults";
import { BatchResultsTable } from "@/components/batch/BatchResultsTable";
import type { BatchProgressDTO } from "@/lib/api";
import { checkerFilterLabel, type CheckerResultFilter } from "@/lib/checkerFilters";

type FilterSummary = {
  count: number;
  confidence: string;
  query: string;
  unopenedOnly: boolean;
  checkerFilter: CheckerResultFilter;
};

type BottomTab = "results" | "log";

type Props = {
  batchThreads: number;
  onBatchThreadsChange: (n: number) => void;
  batchUntestedOnly: boolean;
  onBatchUntestedOnlyChange: (v: boolean) => void;
  batchForceRecheck: boolean;
  onBatchForceRecheckChange: (v: boolean) => void;
  onTestAllFinding?: () => void;
  onTestAllFiltered?: () => void;
  onTestAllQuick?: () => void;
  onTestAllEnvs?: () => void;
  onCancelBatch?: () => void;
  batchRunning?: boolean;
  batchProgress?: BatchProgressDTO | null;
  batchResults: BatchResultRow[];
  batchLogDir?: string;
  onOpenBatchLogs?: () => void;
  batchLogLines: string[];
  onClearBatchLog: () => void;
  onOpenFinding?: (findingId: number) => void;
  filterSummary: FilterSummary;
  hasSelectedFinding?: boolean;
  runningScript?: boolean;
};

export function BatchPanel({
  batchThreads,
  onBatchThreadsChange,
  batchUntestedOnly,
  onBatchUntestedOnlyChange,
  batchForceRecheck,
  onBatchForceRecheckChange,
  onTestAllFinding,
  onTestAllFiltered,
  onTestAllQuick,
  onTestAllEnvs,
  onCancelBatch,
  batchRunning,
  batchProgress,
  batchResults,
  batchLogDir,
  onOpenBatchLogs,
  batchLogLines,
  onClearBatchLog,
  onOpenFinding,
  filterSummary,
  hasSelectedFinding,
  runningScript
}: Props) {
  const busy = batchRunning || !!runningScript;
  const [bottomTab, setBottomTab] = useState<BottomTab>("results");

  useEffect(() => {
    if (batchRunning) setBottomTab("results");
  }, [batchRunning]);

  const filterParts = [
    `${filterSummary.count} findings`,
    filterSummary.confidence || "Todos",
    filterSummary.unopenedOnly ? "só novos" : null,
    batchForceRecheck ? "recheck forçado" : batchUntestedOnly ? "só por testar" : null,
    filterSummary.checkerFilter ? checkerFilterLabel(filterSummary.checkerFilter) : null,
    filterSummary.query ? `«${filterSummary.query}»` : null
  ].filter(Boolean);

  return (
    <div className="flex h-full min-h-0 w-full min-w-0 flex-1 flex-col bg-gs-bg">
      <header className="shrink-0 border-b border-gs-border px-4 py-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-[15px] font-semibold tracking-tight text-gs-fg">Batch</h1>
            <p className="mt-0.5 text-[11px] text-gs-muted">Testar checkers em lote nos findings filtrados</p>
          </div>
          {batchLogDir && onOpenBatchLogs && (
            <button
              type="button"
              className="gs-btn inline-flex items-center gap-1.5 rounded-md text-[11px]"
              onClick={onOpenBatchLogs}
              title={batchLogDir}
            >
              <FolderOpen className="h-3.5 w-3.5" />
              Pasta logs
            </button>
          )}
        </div>

        <div className="mt-3 rounded-md border border-gs-border/70 bg-gs-surface/60 px-3 py-2 text-[11px] text-gs-muted">
          <span className="font-medium text-gs-fg">Filtro activo: </span>
          {filterParts.join(" · ")}
        </div>
      </header>

      <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-4">
        {(batchRunning || batchProgress) && (
          <BatchProgressBar
            progress={batchProgress ?? null}
            running={!!batchRunning}
            onCancel={onCancelBatch}
          />
        )}

        {!batchRunning && (
          <div className="grid shrink-0 gap-2 sm:grid-cols-2 xl:grid-cols-4">
            <ActionCard
              title="Test all envs"
              description="Paralelo com threads."
              icon={Layers}
              disabled={busy}
              onClick={() => onTestAllEnvs?.()}
            />
            <ActionCard
              title="Test all (finding)"
              description={hasSelectedFinding ? "Finding seleccionado." : "Seleccione um finding na lista."}
              icon={FastForward}
              disabled={!hasSelectedFinding || busy}
              onClick={() => onTestAllFinding?.()}
            />
            <ActionCard
              title="Test all (filtro)"
              description="Todos do filtro actual."
              icon={Filter}
              disabled={busy}
              onClick={() => onTestAllFiltered?.()}
            />
            <ActionCard
              title="Test all quick"
              description="Sem email / DB pesado."
              icon={Zap}
              disabled={busy}
              onClick={() => onTestAllQuick?.()}
            />
          </div>
        )}

        <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 rounded-lg border border-gs-border bg-gs-surface/60 px-3 py-2">
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <div className="flex items-center gap-2">
              <span className="w-14 text-[11px] text-gs-muted">Threads</span>
              <div className="inline-flex rounded-md border border-gs-border bg-gs-bg p-0.5">
              {[1, 2, 4, 8, 16].map((n) => (
                <button
                  key={n}
                  type="button"
                  disabled={busy}
                  onClick={() => onBatchThreadsChange(n)}
                  className={clsx(
                    "rounded px-2 py-0.5 text-[11px] tabular-nums",
                    batchThreads === n
                      ? "bg-gs-accent text-white"
                      : "text-gs-muted hover:text-gs-fg"
                  )}
                >
                  {n}
                </button>
              ))}
              </div>
            </div>

          <div className="flex items-center gap-2">
            <span className="w-14 text-[11px] text-gs-muted">Modo</span>
            <div className="inline-flex rounded-md border border-gs-border bg-gs-bg p-0.5">
              <ModeToggle
                active={batchUntestedOnly && !batchForceRecheck}
                disabled={busy}
                onClick={() => {
                  onBatchUntestedOnlyChange(true);
                  onBatchForceRecheckChange(false);
                }}
              >
                Só por testar
              </ModeToggle>
              <ModeToggle
                active={!batchUntestedOnly && !batchForceRecheck}
                disabled={busy}
                onClick={() => {
                  onBatchUntestedOnlyChange(false);
                  onBatchForceRecheckChange(false);
                }}
              >
                Todos
              </ModeToggle>
              <ModeToggle
                active={batchForceRecheck}
                disabled={busy}
                onClick={() => {
                  onBatchUntestedOnlyChange(false);
                  onBatchForceRecheckChange(true);
                }}
              >
                Forçar recheck
              </ModeToggle>
            </div>
          </div>
          </div>

          {busy && !batchRunning && (
            <span className="text-[10px] text-gs-warning">Script a correr — aguarde para iniciar batch</span>
          )}
        </div>

        <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-gs-border bg-gs-surface">
          <div className="flex shrink-0 items-center gap-1 border-b border-gs-border px-2 py-1.5">
            <TabButton active={bottomTab === "results"} onClick={() => setBottomTab("results")}>
              Resultados {batchResults.length > 0 ? `(${batchResults.length})` : ""}
            </TabButton>
            <TabButton active={bottomTab === "log"} onClick={() => setBottomTab("log")}>
              Log {batchLogLines.length > 0 ? `(${batchLogLines.length})` : ""}
            </TabButton>
          </div>

          <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
            {bottomTab === "results" ? (
              <BatchResultsTable
                rows={batchResults}
                running={batchRunning}
                onOpenFinding={onOpenFinding}
              />
            ) : (
              <BatchLogView lines={batchLogLines} onClear={onClearBatchLog} embedded />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function TabButton({
  active,
  onClick,
  children
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={clsx(
        "rounded-md px-2.5 py-1 text-[11px] font-medium transition-colors",
        active ? "bg-gs-accent-muted text-gs-accent" : "text-gs-muted hover:text-gs-fg"
      )}
    >
      {children}
    </button>
  );
}

function ModeToggle({
  active,
  disabled,
  onClick,
  children
}: {
  active: boolean;
  disabled?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={clsx(
        "rounded px-2 py-0.5 text-[11px] whitespace-nowrap",
        active ? "bg-gs-accent text-white" : "text-gs-muted hover:text-gs-fg",
        disabled && "cursor-not-allowed opacity-45"
      )}
    >
      {children}
    </button>
  );
}

function ActionCard({
  title,
  description,
  icon: Icon,
  disabled,
  onClick
}: {
  title: string;
  description: string;
  icon: typeof Layers;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={clsx(
        "group flex items-center gap-2 rounded-lg border px-3 py-2.5 text-left transition-all",
        disabled
          ? "cursor-not-allowed border-gs-border/50 bg-gs-surface/40 opacity-45"
          : "border-gs-border bg-gs-surface hover:border-gs-accent/50 hover:bg-gs-accent-muted/20"
      )}
    >
      <span
        className={clsx(
          "flex h-8 w-8 shrink-0 items-center justify-center rounded-md transition-colors",
          disabled
            ? "bg-gs-surface-raised text-gs-dim"
            : "bg-gs-accent-muted/50 text-gs-accent group-hover:bg-gs-accent group-hover:text-white"
        )}
      >
        <Icon className="h-4 w-4" />
      </span>
      <span className="min-w-0">
        <span className="block truncate text-[11px] font-semibold text-gs-fg">{title}</span>
        <span className="block truncate text-[10px] text-gs-muted">{description}</span>
      </span>
    </button>
  );
}
