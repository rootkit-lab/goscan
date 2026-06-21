import { clsx } from "clsx";
import { CircleStop } from "lucide-react";
import type { BatchProgressDTO } from "@/lib/api";

type Props = {
  progress: BatchProgressDTO | null;
  running: boolean;
  onCancel?: () => void;
};

export function BatchProgressBar({ progress, running, onCancel }: Props) {
  if (!progress && !running) return null;

  const total = progress?.checkTotal ?? 0;
  const done = progress?.checkIndex ?? 0;
  const pct = total > 0 ? Math.min(100, Math.round((done / total) * 100)) : running ? 0 : 100;

  return (
    <div className="shrink-0 rounded-lg border border-gs-accent/30 bg-gs-accent-muted/35 px-4 py-3">
      <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-[13px] font-semibold text-gs-accent">
              {running ? "Batch em curso" : "Batch concluído"}
            </p>
            {total > 0 && (
              <span className="rounded-full bg-gs-surface/80 px-2 py-0.5 text-[11px] tabular-nums text-gs-muted">
                {pct}%
              </span>
            )}
          </div>
          {progress?.domain && (
            <p className="mt-1 truncate text-[12px] text-gs-fg">
              {progress.domain}
              {progress.scriptLabel ? (
                <span className="text-gs-muted"> · {progress.scriptLabel}</span>
              ) : null}
            </p>
          )}
          {progress?.summary && (
            <p className="mt-0.5 truncate text-[11px] text-gs-muted">
              {progress.status} — {progress.summary}
            </p>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <StatPill label="OK" value={progress?.okCount ?? 0} tone="success" />
          <StatPill label="FAIL" value={progress?.failCount ?? 0} tone="error" />
          <StatPill label="SKIP" value={progress?.skipCount ?? 0} tone="muted" />
          {total > 0 && (
            <span className="rounded-md border border-gs-border/60 bg-gs-surface/80 px-2 py-1 text-[11px] tabular-nums text-gs-fg">
              {done}/{total}
              {progress && progress.threads > 1 ? ` · ${progress.threads} threads` : ""}
            </span>
          )}
          {running && onCancel && (
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-md border border-gs-error/50 bg-gs-error/10 px-3 py-1.5 text-[11px] font-medium text-gs-error transition-colors hover:bg-gs-error/20"
              onClick={onCancel}
            >
              <CircleStop className="h-3.5 w-3.5" />
              Parar
            </button>
          )}
        </div>
      </div>

      <div className="h-2.5 overflow-hidden rounded-full bg-gs-border">
        <div
          className={clsx(
            "h-full rounded-full bg-gs-accent transition-[width] duration-300",
            running && pct < 100 && "animate-pulse"
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

function StatPill({
  label,
  value,
  tone
}: {
  label: string;
  value: number;
  tone: "success" | "error" | "muted";
}) {
  return (
    <span
      className={clsx(
        "rounded-md px-2 py-1 text-[11px] tabular-nums",
        tone === "success" && "bg-gs-success/15 text-gs-success",
        tone === "error" && "bg-gs-error/15 text-gs-error",
        tone === "muted" && "bg-gs-surface/80 text-gs-muted"
      )}
    >
      {label} {value}
    </span>
  );
}
