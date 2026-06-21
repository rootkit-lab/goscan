import type { BatchProgressDTO } from "@/lib/api";

type Props = {
  progress: BatchProgressDTO | null;
  running: boolean;
};

export function BatchProgressBar({ progress, running }: Props) {
  if (!running || !progress) return null;

  const total = progress.checkTotal || 0;
  const done = progress.checkIndex || 0;
  const pct = total > 0 ? Math.min(100, Math.round((done / total) * 100)) : 0;

  return (
    <div className="border-b border-vscode-border bg-vscode-sidebar px-2 py-1.5">
      <div className="mb-1 flex items-center justify-between gap-2 text-[10px] text-vscode-muted">
        <span className="truncate">
          Batch {done}/{total}
          {progress.threads > 1 ? ` · ${progress.threads} threads` : ""}
          {progress.domain ? ` · ${progress.domain}` : ""}
        </span>
        <span className="shrink-0 tabular-nums">
          OK {progress.okCount} · FAIL {progress.failCount} · SKIP {progress.skipCount}
        </span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-sm bg-vscode-input">
        <div
          className="h-full bg-vscode-accent transition-[width] duration-200"
          style={{ width: `${pct}%` }}
        />
      </div>
      {progress.scriptLabel && (
        <p className="mt-1 truncate text-[10px] text-vscode-muted">
          {progress.scriptLabel} — {progress.status}
          {progress.summary ? ` · ${progress.summary}` : ""}
        </p>
      )}
    </div>
  );
}
