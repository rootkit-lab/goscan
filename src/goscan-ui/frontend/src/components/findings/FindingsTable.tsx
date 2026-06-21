import {
  ChevronFirst,
  ChevronLast,
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Key,
  Sparkles
} from "lucide-react";
import { clsx } from "clsx";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { FindingCheckerStrip } from "@/components/sidebar/FindingCheckerStrip";
import type { FindingDTO, ScriptCheckerStatusDTO } from "@/lib/api";
import { type CheckerResultFilter } from "@/lib/checkerFilters";
import { formatFindingDate, parseFindingDate } from "@/lib/formatDate";

type SortField = "domain" | "path" | "confidence" | "foundAt" | "modifiedAt";
type SortDir = "asc" | "desc";

const PAGE_SIZES = [25, 50, 100] as const;

function pageRange(page: number, pageSize: number, total: number) {
  if (total === 0) return { from: 0, to: 0 };
  const from = page * pageSize + 1;
  const to = Math.min(total, (page + 1) * pageSize);
  return { from, to };
}

function pageNumbers(current: number, total: number): (number | "…")[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i);
  const pages: (number | "…")[] = [0];
  const start = Math.max(1, current - 1);
  const end = Math.min(total - 2, current + 1);
  if (start > 1) pages.push("…");
  for (let i = start; i <= end; i++) pages.push(i);
  if (end < total - 2) pages.push("…");
  if (total > 1) pages.push(total - 1);
  return pages;
}

function confidenceRank(c: string): number {
  if (c === "HIGH") return 2;
  if (c === "MEDIUM") return 1;
  return 0;
}

function confidenceClass(confidence: string): string {
  if (confidence === "HIGH") return "bg-gs-error/15 text-gs-error";
  if (confidence === "MEDIUM") return "bg-gs-warning/15 text-gs-warning";
  return "bg-gs-surface-raised text-gs-muted";
}

type Props = {
  findings: FindingDTO[];
  selectedId: number | null;
  onSelect: (id: number) => void;
  onOpen: (id: number) => void;
  checkerOverview: Record<number, ScriptCheckerStatusDTO[]>;
  runningScript?: { findingId: number; scriptId: string };
  checkerFilter?: CheckerResultFilter;
};

export function FindingsTable({
  findings,
  selectedId,
  onSelect,
  onOpen,
  checkerOverview,
  runningScript,
  checkerFilter
}: Props) {
  const [sortField, setSortField] = useState<SortField>("foundAt");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState<(typeof PAGE_SIZES)[number]>(50);

  useEffect(() => {
    setPage(0);
  }, [findings.length, sortField, sortDir, pageSize]);

  const sorted = useMemo(() => {
    const list = [...findings];
    const dir = sortDir === "asc" ? 1 : -1;
    list.sort((a, b) => {
      switch (sortField) {
        case "domain":
          return dir * a.domain.localeCompare(b.domain);
        case "path":
          return dir * a.path.localeCompare(b.path);
        case "confidence":
          return dir * (confidenceRank(a.confidence) - confidenceRank(b.confidence));
        case "modifiedAt":
          return dir * (parseFindingDate(a.modifiedAt) - parseFindingDate(b.modifiedAt));
        case "foundAt":
        default:
          return dir * (parseFindingDate(a.foundAt) - parseFindingDate(b.foundAt));
      }
    });
    return list;
  }, [findings, sortField, sortDir]);

  const totalPages = Math.max(1, Math.ceil(sorted.length / pageSize));
  const safePage = Math.min(page, totalPages - 1);
  const pageItems = sorted.slice(safePage * pageSize, safePage * pageSize + pageSize);
  const range = pageRange(safePage, pageSize, sorted.length);
  const pages = pageNumbers(safePage, totalPages);

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDir(field === "domain" || field === "path" ? "asc" : "desc");
    }
  };

  const SortBtn = ({ field, children }: { field: SortField; children: ReactNode }) => (
    <button
      type="button"
      onClick={() => toggleSort(field)}
      className={clsx(
        "inline-flex items-center gap-1 text-left hover:text-gs-fg",
        sortField === field ? "text-gs-accent" : "text-gs-muted"
      )}
    >
      {children}
      {sortField === field && <span className="text-[9px]">{sortDir === "asc" ? "▲" : "▼"}</span>}
    </button>
  );

  if (findings.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center px-6 py-16 text-center">
        <p className="text-[13px] text-gs-fg">Sem resultados</p>
        <p className="mt-1 text-[11px] text-gs-muted">Ajuste filtros ou inicie um scan.</p>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="min-h-0 flex-1 overflow-auto">
        <table className="w-full min-w-[880px] border-collapse text-left text-[12px]">
          <thead className="sticky top-0 z-10 bg-gs-surface shadow-[0_1px_0_var(--gs-border)]">
            <tr className="text-[10px] uppercase tracking-wide">
              <th className="w-8 px-3 py-2" />
              <th className="px-3 py-2 font-medium">
                <SortBtn field="domain">Domínio</SortBtn>
              </th>
              <th className="px-3 py-2 font-medium">
                <SortBtn field="path">Path</SortBtn>
              </th>
              <th className="w-16 px-3 py-2 font-medium">
                <SortBtn field="confidence">Conf.</SortBtn>
              </th>
              <th className="w-36 px-3 py-2 font-medium">
                <SortBtn field="foundAt">Captura</SortBtn>
              </th>
              <th className="w-36 px-3 py-2 font-medium">
                <SortBtn field="modifiedAt">Edição</SortBtn>
              </th>
              <th className="px-3 py-2 font-medium text-gs-muted">Checkers</th>
              <th className="w-24 px-3 py-2 font-medium text-gs-muted">Abrir</th>
            </tr>
          </thead>
          <tbody>
            {pageItems.map((f) => {
              const selected = selectedId === f.id;
              const scripts = checkerOverview[f.id] ?? [];
              const runId = runningScript?.findingId === f.id ? runningScript.scriptId : undefined;
              return (
                <tr
                  key={f.id}
                  onClick={() => onSelect(f.id)}
                  className={clsx(
                    "cursor-pointer border-b border-gs-border/40 transition-colors hover:bg-gs-hover/80",
                    selected && "bg-gs-accent-muted/60"
                  )}
                >
                  <td className="px-3 py-2 text-center">
                    {f.hasCredentials ? (
                      <span title="Credenciais">
                        <Key className="inline h-3.5 w-3.5 text-gs-warning" />
                      </span>
                    ) : f.isNew ? (
                      <span title="Novo">
                        <Sparkles className="inline h-3.5 w-3.5 text-sky-400" />
                      </span>
                    ) : null}
                  </td>
                  <td className="max-w-[220px] truncate px-3 py-2 font-medium text-gs-fg" title={f.domain}>
                    {f.domain}
                  </td>
                  <td className="max-w-[160px] truncate px-3 py-2 text-gs-muted" title={f.path}>
                    {f.path}
                  </td>
                  <td className="px-3 py-2">
                    <span
                      className={clsx(
                        "rounded px-1.5 py-0.5 text-[9px] font-semibold uppercase",
                        confidenceClass(f.confidence)
                      )}
                    >
                      {f.confidence}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 tabular-nums text-gs-muted">
                    {formatFindingDate(f.foundAt)}
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 tabular-nums text-gs-muted">
                    {formatFindingDate(f.modifiedAt)}
                  </td>
                  <td className="px-3 py-2">
                    {f.hasCredentials && scripts.length > 0 ? (
                      <FindingCheckerStrip
                        scripts={scripts}
                        runningScriptId={runId}
                        checkerFilter={checkerFilter}
                        maxVisible={4}
                        compact
                      />
                    ) : (
                      <span className="text-gs-dim">—</span>
                    )}
                  </td>
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      className="gs-btn inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] hover:border-gs-accent/40 hover:bg-gs-accent-muted/40"
                      onClick={(e) => {
                        e.stopPropagation();
                        onOpen(f.id);
                      }}
                    >
                      <ExternalLink className="h-3 w-3" />
                      Abrir
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <footer className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-gs-border bg-gs-surface-raised/50 px-4 py-2.5">
        <p className="text-[11px] text-gs-muted">
          {sorted.length === 0 ? (
            "Sem resultados"
          ) : (
            <>
              <span className="font-medium text-gs-fg">
                {range.from}–{range.to}
              </span>
              <span> de {sorted.length}</span>
              {totalPages > 1 && (
                <span className="hidden sm:inline">
                  {" "}
                  · página {safePage + 1}/{totalPages}
                </span>
              )}
            </>
          )}
        </p>

        {totalPages > 1 && (
          <nav className="flex items-center gap-0.5" aria-label="Paginação">
            <PaginationIconBtn
              disabled={safePage <= 0}
              onClick={() => setPage(0)}
              title="Primeira página"
            >
              <ChevronFirst className="h-3.5 w-3.5" />
            </PaginationIconBtn>
            <PaginationIconBtn
              disabled={safePage <= 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              title="Página anterior"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
            </PaginationIconBtn>

            <span className="mx-1 flex items-center gap-0.5">
              {pages.map((p, i) =>
                p === "…" ? (
                  <span key={`ellipsis-${i}`} className="px-1 text-gs-dim">
                    …
                  </span>
                ) : (
                  <button
                    key={p}
                    type="button"
                    onClick={() => setPage(p)}
                    className={clsx(
                      "min-w-[1.75rem] rounded-md px-1.5 py-1 text-[11px] tabular-nums transition-colors",
                      p === safePage
                        ? "bg-gs-accent font-semibold text-white"
                        : "text-gs-muted hover:bg-gs-hover hover:text-gs-fg"
                    )}
                    aria-current={p === safePage ? "page" : undefined}
                  >
                    {p + 1}
                  </button>
                )
              )}
            </span>

            <PaginationIconBtn
              disabled={safePage >= totalPages - 1}
              onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
              title="Próxima página"
            >
              <ChevronRight className="h-3.5 w-3.5" />
            </PaginationIconBtn>
            <PaginationIconBtn
              disabled={safePage >= totalPages - 1}
              onClick={() => setPage(totalPages - 1)}
              title="Última página"
            >
              <ChevronLast className="h-3.5 w-3.5" />
            </PaginationIconBtn>
          </nav>
        )}

        <div className="inline-flex items-center gap-2">
          <span className="text-[10px] uppercase tracking-wide text-gs-dim">Linhas</span>
          <div className="inline-flex rounded-md border border-gs-border bg-gs-bg p-0.5">
            {PAGE_SIZES.map((n) => (
              <button
                key={n}
                type="button"
                onClick={() => setPageSize(n)}
                className={clsx(
                  "rounded px-2 py-0.5 text-[11px] tabular-nums transition-colors",
                  pageSize === n
                    ? "bg-gs-accent text-white"
                    : "text-gs-muted hover:text-gs-fg"
                )}
              >
                {n}
              </button>
            ))}
          </div>
        </div>
      </footer>
    </div>
  );
}

function PaginationIconBtn({
  disabled,
  onClick,
  title,
  children
}: {
  disabled: boolean;
  onClick: () => void;
  title: string;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      title={title}
      className={clsx(
        "rounded-md p-1.5 text-gs-muted transition-colors",
        disabled ? "cursor-not-allowed opacity-30" : "hover:bg-gs-hover hover:text-gs-fg"
      )}
    >
      {children}
    </button>
  );
}
