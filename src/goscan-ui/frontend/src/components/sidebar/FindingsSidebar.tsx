import { Key, Search, Sparkles } from "lucide-react";
import { FindingCheckerStrip } from "@/components/sidebar/FindingCheckerStrip";
import type { FindingDTO, ScriptCheckerStatusDTO } from "@/lib/api";
import {
  CHECKER_FILTER_OPTIONS,
  countCheckerFilter,
  type CheckerResultFilter
} from "@/lib/checkerFilters";

const CONFIDENCE_OPTIONS = [
  { value: "", label: "All" },
  { value: "HIGH", label: "HIGH" },
  { value: "MEDIUM", label: "MEDIUM" }
];

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
  checkerOverview,
  runningScript
}: Props) {
  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-vscode-border px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-vscode-muted">
        Explorer
      </div>
      <div className="space-y-2 border-b border-vscode-border p-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-vscode-muted" />
          <input
            className="vscode-input pl-7"
            placeholder="Search findings (Ctrl+K)"
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
          />
        </div>
        <div className="flex gap-1">
          {CONFIDENCE_OPTIONS.map((opt) => (
            <button
              key={opt.value || "all"}
              type="button"
              onClick={() => onConfidenceChange(opt.value)}
              className={`flex-1 border px-1 py-0.5 text-[11px] ${
                confidence === opt.value
                  ? "border-vscode-accent bg-vscode-selection text-vscode-fg"
                  : "border-vscode-border bg-transparent text-vscode-muted hover:bg-vscode-hover"
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={() => onUnopenedOnlyChange(!unopenedOnly)}
          className={`flex w-full items-center justify-center gap-1.5 border px-2 py-1 text-[11px] ${
            unopenedOnly
              ? "border-vscode-accent bg-vscode-selection text-vscode-fg"
              : "border-vscode-border bg-transparent text-vscode-muted hover:bg-vscode-hover"
          }`}
          title="Mostrar apenas findings nunca abertos"
        >
          <Sparkles className="h-3 w-3" />
          Novos
          {unopenedCount > 0 && (
            <span
              className={`rounded px-1 text-[10px] ${
                unopenedOnly ? "bg-vscode-accent text-white" : "bg-vscode-input text-vscode-fg"
              }`}
            >
              {unopenedCount}
            </span>
          )}
        </button>
        <div className="max-h-[88px] overflow-y-auto overflow-x-hidden rounded border border-vscode-border/60 p-1">
          <div className="flex flex-wrap gap-1">
          {CHECKER_FILTER_OPTIONS.map((opt) => {
            const active = checkerFilter === opt.value;
            const count = countCheckerFilter(findingIdsForCounts, opt.value, checkerOverview);
            return (
              <button
                key={opt.value}
                type="button"
                title={opt.title}
                onClick={() => onCheckerFilterChange(active ? "" : opt.value)}
                className={`inline-flex shrink-0 items-center justify-center gap-1 border px-1.5 py-0.5 text-[10px] ${
                  active
                    ? "border-emerald-600/60 bg-emerald-950/40 text-emerald-300"
                    : "border-vscode-border bg-transparent text-vscode-muted hover:bg-vscode-hover"
                }`}
              >
                {opt.label}
                {count > 0 && (
                  <span
                    className={`rounded px-1 text-[9px] ${
                      active ? "bg-emerald-700/50 text-emerald-100" : "bg-vscode-input text-vscode-fg"
                    }`}
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
      <div className="min-h-0 flex-1 overflow-auto py-1">
        {findings.length === 0 && (
          <p className="px-3 py-4 text-vscode-muted">
            {checkerFilter ? "Nenhum resultado válido" : "No results"}
          </p>
        )}
        {findings.map((f) => {
          const scripts = checkerOverview[f.id] ?? [];
          const runId = runningScript?.findingId === f.id ? runningScript.scriptId : undefined;
          return (
            <button
              key={f.id}
              type="button"
              onClick={() => onSelect(f.id)}
              className={`flex w-full items-start gap-2 px-2 py-1 text-left hover:bg-vscode-hover ${
                selectedId === f.id ? "bg-vscode-list-active" : ""
              }`}
            >
              {f.hasCredentials ? (
                <Key className="mt-0.5 h-3.5 w-3.5 shrink-0 text-vscode-warning" />
              ) : f.isNew ? (
                <span
                  className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-sky-400"
                  title="Nunca aberto"
                />
              ) : (
                <span className="mt-0.5 inline-block h-3.5 w-3.5 shrink-0" />
              )}
              <span className="min-w-0 flex-1">
                <span className="block truncate text-vscode-fg">{f.domain}</span>
                <span className="block truncate text-[11px] text-vscode-muted">
                  {f.path} · {f.confidence}
                </span>
                {f.hasCredentials && scripts.length > 0 && (
                  <FindingCheckerStrip
                    scripts={scripts}
                    runningScriptId={runId}
                    checkerFilter={checkerFilter}
                  />
                )}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
