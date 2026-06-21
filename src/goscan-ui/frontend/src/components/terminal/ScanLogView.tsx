import { Play, Square } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { clsx } from "clsx";
import { useXtermLog } from "@/hooks/useXtermLog";
import type { ScanWorkerProgressDTO } from "@/lib/api";

type Props = {
  lines: string[];
  filter: string;
  onFilterChange: (v: string) => void;
  workers: ScanWorkerProgressDTO[];
  scanRunning?: boolean;
  onCancelScan?: () => void;
  onRestartScan?: () => void;
  onClear?: () => void;
};

function matchesFilter(line: string, filter: string): boolean {
  if (filter === "all") return true;
  if (filter === "system") return !/^\[[^\]]+\]/.test(line);
  const tag = filter === "local" ? "[Local]" : `[${filter}]`;
  return line.includes(tag);
}

function statusLabel(wp: ScanWorkerProgressDTO) {
  if (wp.status === "running") {
    const base = `${wp.domainsScanned}/${wp.domainsTotal || "?"}`;
    return wp.phaseLabel ? `${base} · ${wp.phaseLabel}` : base;
  }
  if (wp.status === "deploying" && wp.phaseLabel) {
    return `${wp.phaseLabel}${wp.phasePercent != null ? ` ${wp.phasePercent}%` : ""}`;
  }
  return wp.status;
}

export function ScanLogView({
  lines,
  filter,
  onFilterChange,
  workers,
  scanRunning,
  onCancelScan,
  onRestartScan,
  onClear
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const prevCountRef = useRef(0);
  const { writeln, reset } = useXtermLog(containerRef, { enabled: true });

  const filtered = useMemo(() => lines.filter((l) => matchesFilter(l, filter)), [lines, filter]);

  const filterOptions = useMemo(() => {
    const opts = [
      { id: "all", label: "Todos" },
      { id: "system", label: "Sistema" },
      { id: "local", label: "Local" }
    ];
    for (const w of workers) {
      if (w.workerId === "local") continue;
      opts.push({ id: w.workerName, label: w.workerName });
    }
    return opts;
  }, [workers]);

  useEffect(() => {
    if (filtered.length < prevCountRef.current) {
      reset();
      prevCountRef.current = 0;
      for (const line of filtered) writeln(line);
      prevCountRef.current = filtered.length;
      return;
    }
    for (let i = prevCountRef.current; i < filtered.length; i++) {
      writeln(filtered[i] ?? "");
    }
    prevCountRef.current = filtered.length;
  }, [filtered, writeln, reset]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!scanRunning || !onCancelScan) return;
      if (e.ctrlKey && e.key.toLowerCase() === "c") {
        const target = e.target as HTMLElement;
        if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable) return;
        e.preventDefault();
        onCancelScan();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [scanRunning, onCancelScan]);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-gs-border/60 px-2 py-1.5">
        <select
          className="rounded border border-gs-border bg-gs-bg px-2 py-0.5 text-[11px] text-gs-fg"
          value={filter}
          onChange={(e) => onFilterChange(e.target.value)}
        >
          {filterOptions.map((o) => (
            <option key={o.id} value={o.id}>
              {o.label}
            </option>
          ))}
        </select>

        <div className="flex min-w-0 flex-1 flex-wrap gap-1">
          {workers.map((wp) => (
            <button
              key={wp.workerId}
              type="button"
              title={wp.error || wp.status}
              className={clsx(
                "rounded px-1.5 py-0.5 text-[10px] tabular-nums",
                wp.status === "failed" && "bg-gs-error/15 text-gs-error",
                wp.status === "done" && "bg-gs-accent/15 text-gs-accent",
                wp.running && "bg-gs-accent-muted text-gs-accent",
                !wp.running && wp.status !== "failed" && wp.status !== "done" && "bg-gs-hover text-gs-muted"
              )}
              onClick={() => onFilterChange(wp.workerId === "local" ? "local" : wp.workerName)}
            >
              {wp.workerName}: {statusLabel(wp)}
            </button>
          ))}
        </div>

        <span className="flex shrink-0 gap-1">
          {scanRunning ? (
            <button
              type="button"
              className="flex items-center gap-1 rounded px-2 py-0.5 text-[10px] text-gs-error hover:bg-gs-error/10"
              onClick={onCancelScan}
              title="Parar (Ctrl+C)"
            >
              <Square className="h-3 w-3" />
              Parar
            </button>
          ) : (
            <button
              type="button"
              className="flex items-center gap-1 rounded px-2 py-0.5 text-[10px] text-gs-accent hover:bg-gs-accent/10"
              onClick={onRestartScan}
              title="Iniciar scan novamente"
            >
              <Play className="h-3 w-3" />
              Reiniciar
            </button>
          )}
          {onClear && (
            <button type="button" className="rounded px-2 py-0.5 text-[10px] text-gs-muted hover:bg-gs-hover" onClick={onClear}>
              Limpar
            </button>
          )}
        </span>
      </div>

      <div ref={containerRef} className="min-h-0 flex-1 p-1" />

      {scanRunning && (
        <p className="shrink-0 border-t border-gs-border/40 px-2 py-0.5 text-[10px] text-gs-muted">
          Ctrl+C para parar
          {workers.some((w) => w.phaseLabel === "hub")
            ? " · hub WebSocket activo"
            : workers.some((w) => w.phaseLabel === "fallback")
              ? " · fallback stderr/export"
              : ""}
        </p>
      )}
    </div>
  );
}
