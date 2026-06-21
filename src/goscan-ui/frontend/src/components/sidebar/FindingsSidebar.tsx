import { Filter, Search, Sparkles } from "lucide-react";
import { clsx } from "clsx";
import { FindingsTable } from "@/components/findings/FindingsTable";
import type { FindingDTO, ScriptCheckerStatusDTO } from "@/lib/api";
import {
  CHECKER_FILTER_OPTIONS,
  countCheckerFilter,
  type CheckerResultFilter
} from "@/lib/checkerFilters";

const CONFIDENCE_OPTIONS = [
  { value: "", label: "Todos" },
  { value: "HIGH", label: "HIGH" },
  { value: "MEDIUM", label: "MED" }
] as const;

type Props = {
  query: string;
  onQueryChange: (q: string) => void;
  confidence: string;
  onConfidenceChange: (c: string) => void;
  unopenedOnly: boolean;
  onUnopenedOnlyChange: (v: boolean) => void;
  unopenedCount: number;
  checkerFilter: CheckerResultFilter;
  onCheckerFilterChange: (f: CheckerResultFilter) => void;
  findings: FindingDTO[];
  findingIdsForCounts: number[];
  selectedId: number | null;
  onSelect: (id: number) => void;
  onOpen: (id: number) => void;
  checkerOverview: Record<number, ScriptCheckerStatusDTO[]>;
  runningScript?: { findingId: number; scriptId: string };
};

export function FindingsSidebar({
  query,
  onQueryChange,
  confidence,
  onConfidenceChange,
  unopenedOnly,
  onUnopenedOnlyChange,
  unopenedCount,
  checkerFilter,
  onCheckerFilterChange,
  findings,
  findingIdsForCounts,
  selectedId,
  onSelect,
  onOpen,
  checkerOverview,
  runningScript
}: Props) {
  const totalCount = findingIdsForCounts.length;
  const hasActiveFilters = !!checkerFilter || unopenedOnly || !!confidence || !!query;

  return (
    <div className="flex h-full flex-col bg-gs-bg">
      <div className="shrink-0 border-b border-gs-border px-4 py-3">
        <div className="mb-3 flex items-end justify-between gap-3">
          <div>
            <h1 className="text-[15px] font-semibold tracking-tight text-gs-fg">Findings</h1>
            <p className="mt-0.5 text-[11px] text-gs-muted">
              {findings.length === totalCount
                ? `${totalCount} resultados`
                : `${findings.length} de ${totalCount} resultados`}
              {hasActiveFilters ? " · filtrado" : ""}
            </p>
          </div>
          {unopenedCount > 0 && (
            <span className="shrink-0 rounded-full bg-sky-500/15 px-2 py-0.5 text-[10px] font-medium text-sky-300">
              {unopenedCount} novos
            </span>
          )}
        </div>

        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gs-muted" />
          <input
            className="gs-input w-full rounded-md pl-9"
            placeholder="Pesquisar domínio ou path… (Ctrl+K)"
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
          />
        </div>

        <div className="mt-3 flex flex-wrap items-center gap-2">
          <div className="inline-flex rounded-md border border-gs-border bg-gs-surface p-0.5">
            {CONFIDENCE_OPTIONS.map((opt) => (
              <button
                key={opt.value || "all"}
                type="button"
                onClick={() => onConfidenceChange(opt.value)}
                className={clsx(
                  "rounded px-2.5 py-1 text-[11px] font-medium transition-colors",
                  confidence === opt.value
                    ? "bg-gs-accent text-white shadow-sm"
                    : "text-gs-muted hover:text-gs-fg"
                )}
              >
                {opt.label}
              </button>
            ))}
          </div>

          <button
            type="button"
            onClick={() => onUnopenedOnlyChange(!unopenedOnly)}
            className={clsx(
              "inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] font-medium transition-colors",
              unopenedOnly
                ? "border-sky-500/40 bg-sky-500/10 text-sky-300"
                : "border-gs-border bg-gs-surface text-gs-muted hover:border-gs-border hover:bg-gs-hover hover:text-gs-fg"
            )}
            title="Mostrar apenas findings nunca abertos"
          >
            <Sparkles className="h-3 w-3" />
            Novos
            {unopenedCount > 0 && (
              <span className="rounded-full bg-gs-surface-raised px-1.5 text-[10px] tabular-nums">{unopenedCount}</span>
            )}
          </button>
        </div>

        <div className="mt-2.5">
          <div className="mb-1.5 flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wide text-gs-muted">
            <Filter className="h-3 w-3" />
            Checkers válidos
          </div>
          <div className="flex flex-wrap gap-1.5">
            {CHECKER_FILTER_OPTIONS.map((opt) => {
              const active = checkerFilter === opt.value;
              const count = countCheckerFilter(findingIdsForCounts, opt.value, checkerOverview);
              return (
                <button
                  key={opt.value}
                  type="button"
                  title={opt.title}
                  onClick={() => onCheckerFilterChange(active ? "" : opt.value)}
                  className={clsx(
                    "inline-flex shrink-0 items-center gap-1 rounded-full border px-2.5 py-1 text-[11px] transition-colors",
                    active
                      ? "border-gs-success/40 bg-gs-success/10 text-gs-success"
                      : "border-gs-border/80 bg-gs-surface text-gs-muted hover:border-gs-border hover:bg-gs-hover hover:text-gs-fg"
                  )}
                >
                  {opt.label}
                  {count > 0 && (
                    <span
                      className={clsx(
                        "min-w-[1.1rem] rounded-full px-1 text-center text-[9px] tabular-nums",
                        active ? "bg-gs-success/20 text-gs-success" : "bg-gs-surface-raised text-gs-fg"
                      )}
                    >
                      {count}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      </div>

      <FindingsTable
        findings={findings}
        selectedId={selectedId}
        onSelect={onSelect}
        onOpen={onOpen}
        checkerOverview={checkerOverview}
        runningScript={runningScript}
        checkerFilter={checkerFilter}
      />
    </div>
  );
}
