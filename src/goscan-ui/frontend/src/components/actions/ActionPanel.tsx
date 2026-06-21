import { Play, Square } from "lucide-react";
import { ScriptList } from "@/components/actions/ScriptList";
import { CollapsibleSection } from "@/components/layout/CollapsibleSection";
import { SettingsPanel } from "@/components/settings/SettingsPanel";
import type { BatchProgressDTO, ScanOptsDTO, ScriptCheckerStatusDTO, SettingsDTO } from "@/lib/api";

type Props = {
  scripts: ScriptCheckerStatusDTO[];
  selectedScript: string;
  onScriptChange: (id: string) => void;
  onRunScript: (scriptId?: string) => void;
  scanOpts: ScanOptsDTO & { threads: number; fast: boolean; rescan: boolean; timeoutSec: number };
  onScanOptsChange: (opts: Props["scanOpts"]) => void;
  onStartScan: () => void;
  onCancelScan: () => void;
  onCancelScript?: () => void;
  onTestAllFinding?: () => void;
  onTestAllFiltered?: () => void;
  onTestAllQuick?: () => void;
  onTestAllEnvs?: () => void;
  onCancelBatch?: () => void;
  batchRunning?: boolean;
  batchProgress?: BatchProgressDTO | null;
  batchLabel?: string;
  batchThreads: number;
  onBatchThreadsChange: (n: number) => void;
  terminalActive?: boolean;
  runningScriptId?: string;
  settings: SettingsDTO | null;
  draftDataDir: string;
  draftScanDir: string;
  onDraftDataDirChange: (v: string) => void;
  onDraftScanDirChange: (v: string) => void;
  onPickDataDir: () => void;
  onPickScanDir: () => void;
  onSaveSettings: () => void;
  onOpenDataDir: () => void;
  onOpenScanDir: () => void;
  settingsSaving?: boolean;
};

export function ActionPanel({
  scripts,
  selectedScript,
  onScriptChange,
  onRunScript,
  scanOpts,
  onScanOptsChange,
  onStartScan,
  onCancelScan,
  onCancelScript,
  onTestAllFinding,
  onTestAllFiltered,
  onTestAllQuick,
  onTestAllEnvs,
  onCancelBatch,
  batchRunning,
  batchProgress,
  batchLabel,
  batchThreads,
  onBatchThreadsChange,
  terminalActive,
  runningScriptId,
  settings,
  draftDataDir,
  draftScanDir,
  onDraftDataDirChange,
  onDraftScanDirChange,
  onPickDataDir,
  onPickScanDir,
  onSaveSettings,
  onOpenDataDir,
  onOpenScanDir,
  settingsSaving
}: Props) {
  return (
    <div className="flex h-full flex-col overflow-auto">
      <div className="border-b border-vscode-border px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-vscode-muted">
        Actions
      </div>

      <SettingsPanel
        settings={settings}
        draftDataDir={draftDataDir}
        draftScanDir={draftScanDir}
        onDraftDataDirChange={onDraftDataDirChange}
        onDraftScanDirChange={onDraftScanDirChange}
        onPickDataDir={onPickDataDir}
        onPickScanDir={onPickScanDir}
        onSave={onSaveSettings}
        onOpenDataDir={onOpenDataDir}
        onOpenScanDir={onOpenScanDir}
        saving={settingsSaving}
      />

      <CollapsibleSection title="Scripts">
        <ScriptList
          scripts={scripts}
          selectedScript={selectedScript}
          onScriptChange={onScriptChange}
          onRunScript={onRunScript}
          onCancelScript={onCancelScript}
          onTestAllFinding={onTestAllFinding}
          onTestAllFiltered={onTestAllFiltered}
          onTestAllQuick={onTestAllQuick}
          onTestAllEnvs={onTestAllEnvs}
          onCancelBatch={onCancelBatch}
          batchRunning={batchRunning}
          batchProgress={batchProgress}
          batchLabel={batchLabel}
          batchThreads={batchThreads}
          onBatchThreadsChange={onBatchThreadsChange}
          terminalActive={terminalActive}
          runningScriptId={runningScriptId}
        />
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
