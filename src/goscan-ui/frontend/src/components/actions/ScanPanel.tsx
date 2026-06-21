import { FolderOpen, Play, Square } from "lucide-react";
import { clsx } from "clsx";
import type { ScanProgressDTO } from "@/lib/api";

type ScanOpts = {
  threads: number;
  fast: boolean;
  rescan: boolean;
  timeoutSec: number;
};

type Props = {
  scanOpts: ScanOpts;
  onScanOptsChange: (opts: ScanOpts) => void;
  onStartScan: () => void;
  onCancelScan: () => void;
  scanRunning?: boolean;
  scanProgress?: ScanProgressDTO | null;
  scanDir?: string;
};

export function ScanPanel({
  scanOpts,
  onScanOptsChange,
  onStartScan,
  onCancelScan,
  scanRunning,
  scanProgress,
  scanDir
}: Props) {
  const shortDir = scanDir
    ? scanDir.length > 42
      ? "…" + scanDir.slice(-41)
      : scanDir
    : undefined;

  return (
    <section className="flex h-full shrink-0 flex-col border-l border-gs-border bg-gs-surface">
      <div className="border-b border-gs-border px-4 py-3">
        <h2 className="text-[13px] font-semibold text-gs-fg">Scan</h2>
        <p className="mt-0.5 text-[11px] text-gs-muted">Descobrir .env expostos</p>
      </div>

      <div className="flex flex-1 flex-col gap-4 overflow-auto px-4 py-4">
        {scanRunning && (
          <div className="rounded-lg border border-gs-accent/30 bg-gs-accent-muted/40 p-3">
            <div className="mb-2 flex items-center justify-between gap-2 text-[11px]">
              <span className="font-medium text-gs-accent">Scan em curso</span>
              {scanProgress && (
                <span className="tabular-nums text-gs-muted">
                  {scanProgress.domainsScanned} dom · {scanProgress.vulnsFound} vulns
                </span>
              )}
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-gs-border/80">
              <div className="h-full w-full animate-pulse rounded-full bg-gs-accent" />
            </div>
            {scanProgress && scanProgress.domainsPending > 0 && (
              <p className="mt-2 text-[10px] text-gs-muted">{scanProgress.domainsPending} domínios pendentes</p>
            )}
          </div>
        )}

        {shortDir && (
          <div
            className="flex items-start gap-2 rounded-md border border-gs-border/60 bg-gs-bg/50 px-2.5 py-2"
            title={scanDir}
          >
            <FolderOpen className="mt-0.5 h-3.5 w-3.5 shrink-0 text-gs-muted" />
            <span className="min-w-0 break-all text-[10px] leading-relaxed text-gs-muted">{shortDir}</span>
          </div>
        )}

        <div className="space-y-3">
          <label className="block">
            <span className="mb-1.5 block text-[11px] font-medium text-gs-muted">Threads</span>
            <input
              type="number"
              min={1}
              max={200}
              className="gs-input w-full rounded-md"
              value={scanOpts.threads}
              disabled={scanRunning}
              onChange={(e) => onScanOptsChange({ ...scanOpts, threads: Number(e.target.value) || 1 })}
            />
          </label>

          <ToggleRow
            label="Fast paths only"
            hint="Ignorar paths lentos"
            checked={scanOpts.fast}
            disabled={scanRunning}
            onChange={(fast) => onScanOptsChange({ ...scanOpts, fast })}
          />
          <ToggleRow
            label="Rescan scanned"
            hint="Reprocessar domínios já vistos"
            checked={scanOpts.rescan}
            disabled={scanRunning}
            onChange={(rescan) => onScanOptsChange({ ...scanOpts, rescan })}
          />
        </div>
      </div>

      <div className="shrink-0 border-t border-gs-border p-4">
        <div className="flex gap-2">
          <button
            type="button"
            className="gs-btn gs-btn-primary flex flex-1 items-center justify-center gap-2 rounded-md py-2 text-[12px] font-medium"
            disabled={scanRunning}
            onClick={onStartScan}
          >
            <Play className="h-4 w-4" />
            Iniciar scan
          </button>
          <button
            type="button"
            className={clsx(
              "gs-btn flex items-center justify-center rounded-md px-3",
              scanRunning && "border-gs-error/40 text-gs-error hover:bg-gs-error/10"
            )}
            disabled={!scanRunning}
            onClick={onCancelScan}
            title="Parar scan"
          >
            <Square className="h-4 w-4" />
          </button>
        </div>
      </div>
    </section>
  );
}

function ToggleRow({
  label,
  hint,
  checked,
  disabled,
  onChange
}: {
  label: string;
  hint: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label
      className={clsx(
        "flex cursor-pointer items-start gap-3 rounded-md border px-3 py-2 transition-colors",
        checked ? "border-gs-accent/30 bg-gs-accent-muted/30" : "border-gs-border/60 hover:bg-gs-hover/50",
        disabled && "cursor-not-allowed opacity-50"
      )}
    >
      <input
        type="checkbox"
        className="mt-0.5 accent-[var(--gs-accent)]"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span>
        <span className="block text-[12px] text-gs-fg">{label}</span>
        <span className="block text-[10px] text-gs-muted">{hint}</span>
      </span>
    </label>
  );
}
