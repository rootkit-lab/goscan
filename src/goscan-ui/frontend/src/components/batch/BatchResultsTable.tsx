import { ChevronFirst, ChevronLast, ChevronLeft, ChevronRight, ExternalLink } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { CheckerStatusIcon } from "@/components/checkers/CheckerStatusIcon";
import type { BatchResultRow } from "@/components/batch/batchResults";
import { scriptIcon, type CheckerStatus } from "@/lib/scriptIcons";

const PAGE_SIZES = [25, 50, 100] as const;

function statusTone(status: string): CheckerStatus {
  if (status === "ok") return "ok";
  if (status === "skip") return "skip";
  return "fail";
}

type Props = {
  rows: BatchResultRow[];
  running?: boolean;
  onOpenFinding?: (findingId: number) => void;
};

export function BatchResultsTable({ rows, running, onOpenFinding }: Props) {
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState<(typeof PAGE_SIZES)[number]>(50);
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  useEffect(() => {
    if (running) setPage(Math.max(0, Math.ceil(rows.length / pageSize) - 1));
  }, [rows.length, running, pageSize]);

  useEffect(() => {
    setPage(0);
  }, [pageSize, sortDir]);

  const sorted = useMemo(() => {
    const list = [...rows];
    list.sort((a, b) => {
      const cmp = a.checkIndex - b.checkIndex;
      return sortDir === "asc" ? cmp : -cmp;
    });
    return list;
  }, [rows, sortDir]);

  const totalPages = Math.max(1, Math.ceil(sorted.length / pageSize));
  const safePage = Math.min(page, totalPages - 1);
  const pageItems = sorted.slice(safePage * pageSize, safePage * pageSize + pageSize);
  const from = sorted.length === 0 ? 0 : safePage * pageSize + 1;
  const to = Math.min(sorted.length, (safePage + 1) * pageSize);

  if (rows.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 py-16 text-center">
        <p className="text-[13px] text-gs-fg">
          {running ? "A aguardar primeiros resultados…" : "Nenhum resultado de batch ainda."}
        </p>
        {!running && (
          <p className="max-w-sm text-[11px] text-gs-muted">
            Escolha uma acção acima para iniciar. Os resultados aparecem aqui em tempo real.
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <div className="min-h-0 flex-1 overflow-auto overscroll-contain">
        <table className="w-full min-w-[720px] border-collapse text-left text-[12px]">
          <thead className="sticky top-0 z-10 bg-gs-surface shadow-[0_1px_0_var(--gs-border)]">
            <tr className="text-[10px] uppercase tracking-wide text-gs-muted">
              <th className="w-12 px-3 py-2 font-medium">#</th>
              <th className="px-3 py-2 font-medium">Domínio</th>
              <th className="px-3 py-2 font-medium">Checker</th>
              <th className="w-20 px-3 py-2 font-medium">Estado</th>
              <th className="px-3 py-2 font-medium">Resumo</th>
              <th className="w-20 px-3 py-2 font-medium">Abrir</th>
            </tr>
          </thead>
          <tbody>
            {pageItems.map((r) => {
              const status = statusTone(r.status);
              const Icon = scriptIcon(r.scriptId);
              return (
                <tr key={r.id} className="border-b border-gs-border/40 hover:bg-gs-hover/50">
                  <td className="px-3 py-2 tabular-nums text-gs-muted">{r.checkIndex}</td>
                  <td className="max-w-[200px] truncate px-3 py-2 font-medium text-gs-fg">{r.domain}</td>
                  <td className="px-3 py-2">
                    <span className="inline-flex items-center gap-1.5 text-gs-muted">
                      <Icon className="h-3.5 w-3.5 shrink-0" />
                      <span className="truncate">{r.scriptLabel}</span>
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    <CheckerStatusIcon status={status} />
                  </td>
                  <td className="max-w-[280px] truncate px-3 py-2 text-gs-muted">{r.summary || "—"}</td>
                  <td className="px-3 py-2">
                    {onOpenFinding && (
                      <button
                        type="button"
                        className="gs-btn inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px]"
                        onClick={() => onOpenFinding(r.findingId)}
                      >
                        <ExternalLink className="h-3 w-3" />
                        Abrir
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <footer className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-t border-gs-border bg-gs-surface-raised/50 px-3 py-2 text-[11px] text-gs-muted">
        <span>
          <span className="font-medium text-gs-fg">
            {from}–{to}
          </span>{" "}
          de {sorted.length}
        </span>
        <div className="flex items-center gap-1">
          <button
            type="button"
            className="rounded p-1 hover:bg-gs-hover disabled:opacity-30"
            disabled={safePage <= 0}
            onClick={() => setPage(0)}
          >
            <ChevronFirst className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            className="rounded p-1 hover:bg-gs-hover disabled:opacity-30"
            disabled={safePage <= 0}
            onClick={() => setPage((p) => p - 1)}
          >
            <ChevronLeft className="h-3.5 w-3.5" />
          </button>
          <span className="px-2 tabular-nums">
            {safePage + 1}/{totalPages}
          </span>
          <button
            type="button"
            className="rounded p-1 hover:bg-gs-hover disabled:opacity-30"
            disabled={safePage >= totalPages - 1}
            onClick={() => setPage((p) => p + 1)}
          >
            <ChevronRight className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            className="rounded p-1 hover:bg-gs-hover disabled:opacity-30"
            disabled={safePage >= totalPages - 1}
            onClick={() => setPage(totalPages - 1)}
          >
            <ChevronLast className="h-3.5 w-3.5" />
          </button>
          <select
            className="gs-input ml-2 w-auto rounded-md py-0.5 text-[11px]"
            value={pageSize}
            onChange={(e) => setPageSize(Number(e.target.value) as (typeof PAGE_SIZES)[number])}
          >
            {PAGE_SIZES.map((n) => (
              <option key={n} value={n}>
                {n}/pág
              </option>
            ))}
          </select>
          <button
            type="button"
            className="ml-1 rounded px-2 py-0.5 text-[10px] hover:bg-gs-hover"
            onClick={() => setSortDir((d) => (d === "asc" ? "desc" : "asc"))}
          >
            Ordem {sortDir === "asc" ? "↑" : "↓"}
          </button>
        </div>
      </footer>
    </div>
  );
}
