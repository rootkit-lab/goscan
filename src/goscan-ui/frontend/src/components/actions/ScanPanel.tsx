import { FolderOpen, Play, Square } from "lucide-react";
import { clsx } from "clsx";
import type { RemoteWorkerDTO, ScanProgressDTO, ScanWorkerProgressDTO } from "@/lib/api";

type ScanOpts = {
  threads: number;
  fast: boolean;
  rescan: boolean;
  timeoutSec: number;
  targets: string[];
  deployRemote: boolean;
};

type Props = {
  scanOpts: ScanOpts;
  onScanOptsChange: (opts: ScanOpts) => void;
  onStartScan: () => void;
  onCancelScan: () => void;
  scanRunning?: boolean;
  scanProgress?: ScanProgressDTO | null;
  workerProgress?: ScanWorkerProgressDTO[];
  workers?: RemoteWorkerDTO[];
  scanDir?: string;
};

function statusLabel(status: string) {
  switch (status) {
    case "preparing":
      return "a preparar…";
    case "deploying":
      return "a instalar…";
    case "running":
      return "a scanear…";
    case "failed":
      return "falhou";
	case "done":
      return "concluído";
    case "cancelled":
      return "cancelado";
    default:
      return status;
  }
}

export function ScanPanel({
  scanOpts,
  onScanOptsChange,
  onStartScan,
  onCancelScan,
  scanRunning,
  scanProgress,
  workerProgress,
  workers = [],
  scanDir
}: Props) {
  const shortDir = scanDir
    ? scanDir.length > 42
      ? "…" + scanDir.slice(-41)
      : scanDir
    : undefined;

  const toggleTarget = (id: string) => {
    const set = new Set(scanOpts.targets);
    if (set.has(id)) set.delete(id);
    else set.add(id);
    onScanOptsChange({ ...scanOpts, targets: Array.from(set) });
  };

  const ensureDefaultTargets = () => {
    if (scanOpts.targets.length > 0) return;
    const defaults = ["local", ...workers.filter((w) => w.enabled).map((w) => w.id)];
    onScanOptsChange({ ...scanOpts, targets: defaults });
  };

  return (
    <section className="flex h-full shrink-0 flex-col border-l border-gs-border bg-gs-surface">
      <div className="border-b border-gs-border px-4 py-3">
        <h2 className="text-[13px] font-semibold text-gs-fg">Scan</h2>
        <p className="mt-0.5 text-[11px] text-gs-muted">Fila central no goscan; cada filho pede o próximo lote ao terminar</p>
      </div>

      <div className="flex flex-1 flex-col gap-4 overflow-auto px-4 py-4">
        <div>
          <div className="mb-2 flex items-center justify-between">
            <span className="text-[11px] font-medium text-gs-muted">Destinos</span>
            {!scanRunning && scanOpts.targets.length === 0 && (
              <button type="button" className="text-[10px] text-gs-accent hover:underline" onClick={ensureDefaultTargets}>
                Seleccionar todos
              </button>
            )}
          </div>
          <div className="space-y-1.5">
            <label className="flex cursor-pointer items-center gap-2 rounded-md border border-gs-border/60 px-2.5 py-2 text-[11px]">
              <input
                type="checkbox"
                className="accent-[var(--gs-accent)]"
                checked={scanOpts.targets.includes("local")}
                disabled={scanRunning}
                onChange={() => toggleTarget("local")}
              />
              <span>Local (esta máquina)</span>
            </label>
            {workers.map((w) => (
              <label
                key={w.id}
                className={clsx(
                  "flex cursor-pointer items-center gap-2 rounded-md border px-2.5 py-2 text-[11px]",
                  w.enabled ? "border-gs-border/60" : "border-gs-border/30 opacity-60"
                )}
              >
                <input
                  type="checkbox"
                  className="accent-[var(--gs-accent)]"
                  checked={scanOpts.targets.includes(w.id)}
                  disabled={scanRunning || !w.enabled}
                  onChange={() => toggleTarget(w.id)}
                />
                <span className="min-w-0 flex-1 truncate">{w.name || w.host}</span>
                {!w.enabled && <span className="text-[10px] text-gs-muted">offline</span>}
              </label>
            ))}
          </div>
        </div>

        {(scanRunning || (workerProgress && workerProgress.length > 0)) && (
          <div className="rounded-lg border border-gs-accent/30 bg-gs-accent-muted/40 p-3">
            <div className="mb-2 flex items-center justify-between gap-2 text-[11px]">
              <span className="font-medium text-gs-accent">{scanRunning ? "Scan em curso" : "Último scan"}</span>
              {scanProgress && (
                <span className="tabular-nums text-gs-muted">
                  {scanProgress.running
                    ? `${scanProgress.domainsScanned.toLocaleString()} sessão · ${scanProgress.domainsPending.toLocaleString()} fila`
                    : `${scanProgress.domainsScanned} dom · ${scanProgress.vulnsFound} vulns`}
                </span>
              )}
            </div>
            {scanRunning && (
              <p className="mb-2 text-[10px] text-gs-muted">Log detalhado na barra inferior (Scan)</p>
            )}
            {workerProgress && workerProgress.length > 0 ? (
              <div className="space-y-2">
                {workerProgress.map((wp) => {
                  const deployPct =
                    wp.status === "deploying" && wp.phasePercent != null && wp.phasePercent > 0
                      ? wp.phasePercent
                      : null;
                  const scanPct =
                    wp.domainsTotal > 0
                      ? Math.min(100, (wp.domainsScanned / wp.domainsTotal) * 100)
                      : wp.status === "done"
                        ? 100
                        : null;
                  const barPct = deployPct ?? scanPct ?? (wp.status === "preparing" ? 8 : 20);
                  const statusText =
                    wp.status === "deploying" && wp.phaseLabel
                      ? `${wp.phaseLabel}${wp.phasePercent != null ? ` ${wp.phasePercent}%` : ""}`
                      : wp.domainsTotal > 0 &&
                          (wp.status === "running" || wp.status === "done" || wp.status === "preparing")
                        ? `${wp.domainsScanned}/${wp.domainsTotal} neste lote · ${wp.vulnsFound} vulns`
                        : statusLabel(wp.status);
                  return (
                  <div key={wp.workerId}>
                    <div className="mb-0.5 flex justify-between text-[10px] text-gs-muted">
                      <span>{wp.workerName}</span>
                      <span className="tabular-nums">{statusText}</span>
                    </div>
                    <div className="h-1 overflow-hidden rounded-full bg-gs-border/80">
                      <div
                        className={clsx(
                          "h-full rounded-full",
                          wp.status === "failed" ? "bg-gs-error" : "bg-gs-accent",
                          wp.status === "running" && "transition-[width] duration-300"
                        )}
                        style={{ width: `${barPct}%` }}
                      />
                    </div>
                    {wp.error && <p className="mt-0.5 text-[10px] text-gs-error">{wp.error}</p>}
                  </div>
                  );
                })}
              </div>
            ) : (
              <div className="h-1.5 overflow-hidden rounded-full bg-gs-border/80">
                <div className="h-full w-full animate-pulse rounded-full bg-gs-accent" />
              </div>
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

        <label className="flex cursor-pointer items-start gap-2 rounded-md border border-gs-border/60 px-3 py-2 text-[11px]">
          <input
            type="checkbox"
            className="mt-0.5 accent-[var(--gs-accent)]"
            checked={scanOpts.deployRemote}
            disabled={scanRunning}
            onChange={(e) => onScanOptsChange({ ...scanOpts, deployRemote: e.target.checked })}
          />
          <span>
            <span className="block text-gs-fg">Forçar actualização do binário remoto</span>
            <span className="block text-[10px] text-gs-muted">
              Instalação automática se em falta; com checkbox reenvia se versão diferente
            </span>
          </span>
        </label>

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
