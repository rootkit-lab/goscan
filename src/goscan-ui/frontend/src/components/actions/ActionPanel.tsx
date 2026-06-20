import { Play, Square } from "lucide-react";
import type { ScanOptsDTO, ScriptDTO } from "@/lib/api";
import { CollapsibleSection } from "@/components/layout/CollapsibleSection";

type Props = {
  scripts: ScriptDTO[];
  selectedScript: string;
  onScriptChange: (id: string) => void;
  onRunScript: () => void;
  canRunScript: boolean;
  scanOpts: ScanOptsDTO & { threads: number; fast: boolean; rescan: boolean; timeoutSec: number };
  onScanOptsChange: (opts: Props["scanOpts"]) => void;
  onStartScan: () => void;
  onCancelScan: () => void;
  onCancelScript?: () => void;
  terminalActive?: boolean;
};

export function ActionPanel({
  scripts,
  selectedScript,
  onScriptChange,
  onRunScript,
  canRunScript,
  scanOpts,
  onScanOptsChange,
  onStartScan,
  onCancelScan,
  onCancelScript,
  terminalActive
}: Props) {
  return (
    <div className="flex h-full flex-col overflow-auto">
      <div className="border-b border-vscode-border px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-vscode-muted">
        Actions
      </div>

      <CollapsibleSection title="Scripts">
        <label className="mb-1 block text-[11px] text-vscode-muted">Checker</label>
        <select
          className="vscode-input mb-2"
          value={selectedScript}
          onChange={(e) => onScriptChange(e.target.value)}
        >
          {scripts.length === 0 && <option value="">(none compatible)</option>}
          {scripts.map((s) => (
            <option key={s.id} value={s.id}>
              {s.label}
            </option>
          ))}
        </select>
        <div className="flex gap-1">
          <button type="button" className="vscode-btn vscode-btn-primary flex flex-1 items-center justify-center gap-1" disabled={!canRunScript} onClick={onRunScript}>
            <Play className="h-3.5 w-3.5" />
            Run script
          </button>
          {terminalActive && onCancelScript && (
            <button type="button" className="vscode-btn flex items-center justify-center px-2" onClick={onCancelScript} title="Parar script">
              <Square className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </CollapsibleSection>

      <CollapsibleSection title="Scan">
        <label className="mb-1 block text-[11px] text-vscode-muted">Threads</label>
        <input
          type="number"
          className="vscode-input mb-2"
          value={scanOpts.threads}
          onChange={(e) => onScanOptsChange({ ...scanOpts, threads: Number(e.target.value) })}
        />
        <label className="mb-2 flex items-center gap-2 text-[12px] text-vscode-fg">
          <input
            type="checkbox"
            checked={scanOpts.fast}
            onChange={(e) => onScanOptsChange({ ...scanOpts, fast: e.target.checked })}
          />
          Fast paths only
        </label>
        <label className="mb-3 flex items-center gap-2 text-[12px] text-vscode-fg">
          <input
            type="checkbox"
            checked={scanOpts.rescan}
            onChange={(e) => onScanOptsChange({ ...scanOpts, rescan: e.target.checked })}
          />
          Rescan scanned domains
        </label>
        <div className="flex gap-1">
          <button type="button" className="vscode-btn vscode-btn-primary flex flex-1 items-center justify-center gap-1" onClick={onStartScan}>
            <Play className="h-3.5 w-3.5" />
            Start
          </button>
          <button type="button" className="vscode-btn flex items-center justify-center px-2" onClick={onCancelScan} title="Stop scan">
            <Square className="h-3.5 w-3.5" />
          </button>
        </div>
      </CollapsibleSection>
    </div>
  );
}
