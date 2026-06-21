import { FolderOpen, Save } from "lucide-react";
import { CollapsibleSection } from "@/components/layout/CollapsibleSection";
import type { SettingsDTO } from "@/lib/api";

type Props = {
  settings: SettingsDTO | null;
  draftDataDir: string;
  draftScanDir: string;
  onDraftDataDirChange: (v: string) => void;
  onDraftScanDirChange: (v: string) => void;
  onPickDataDir: () => void;
  onPickScanDir: () => void;
  onSave: () => void;
  onOpenDataDir: () => void;
  onOpenScanDir: () => void;
  saving?: boolean;
};

function shortPath(p: string, max = 42) {
  if (!p) return "—";
  if (p.length <= max) return p;
  return "…" + p.slice(-(max - 1));
}

export function SettingsPanel({
  settings,
  draftDataDir,
  draftScanDir,
  onDraftDataDirChange,
  onDraftScanDirChange,
  onPickDataDir,
  onPickScanDir,
  onSave,
  onOpenDataDir,
  onOpenScanDir,
  saving
}: Props) {
  return (
    <CollapsibleSection title="Pastas" defaultOpen={settings?.needsSetup}>
      {settings?.needsSetup && (
        <p className="mb-2 rounded border border-amber-700/50 bg-amber-950/40 px-2 py-1.5 text-[11px] text-amber-100">
          Configure pastas de produção (separadas do repo de dev). Corra{" "}
          <code className="text-amber-50">make migrate-prod-data</code> para copiar dados existentes.
        </p>
      )}
      {settings?.pointsToDevRepo && (
        <p className="mb-2 rounded border border-red-800/50 bg-red-950/40 px-2 py-1.5 text-[11px] text-red-100">
          Atenção: prod aponta para o repo de dev. Use{" "}
          <code className="text-red-50">{settings.defaultProdDataDir || "~/.local/share/goscan/data"}</code>.
        </p>
      )}
      <label className="mb-1 block text-[11px] text-vscode-muted">Dados (DB, findings, logs)</label>
      <div className="mb-1 flex gap-1">
        <input
          className="vscode-input min-w-0 flex-1 text-[11px]"
          value={draftDataDir}
          onChange={(e) => onDraftDataDirChange(e.target.value)}
          placeholder={settings?.defaultProdDataDir || "/caminho/para/dados"}
          title={draftDataDir}
        />
        <button type="button" className="vscode-btn shrink-0 px-2" onClick={onPickDataDir} title="Escolher pasta">
          …
        </button>
        <button type="button" className="vscode-btn shrink-0 px-2" onClick={onOpenDataDir} title="Abrir no gestor">
          <FolderOpen className="h-3.5 w-3.5" />
        </button>
      </div>
      <p className="mb-2 text-[10px] text-vscode-muted">{shortPath(draftDataDir, 56)}</p>

      <label className="mb-1 block text-[11px] text-vscode-muted">Domínios para scan (.txt, .env)</label>
      <div className="mb-1 flex gap-1">
        <input
          className="vscode-input min-w-0 flex-1 text-[11px]"
          value={draftScanDir}
          onChange={(e) => onDraftScanDirChange(e.target.value)}
          placeholder="(default: dados/files)"
          title={draftScanDir}
        />
        <button type="button" className="vscode-btn shrink-0 px-2" onClick={onPickScanDir} title="Escolher pasta">
          …
        </button>
        <button type="button" className="vscode-btn shrink-0 px-2" onClick={onOpenScanDir} title="Abrir no gestor">
          <FolderOpen className="h-3.5 w-3.5" />
        </button>
      </div>
      <p className="mb-2 text-[10px] text-vscode-muted">{shortPath(draftScanDir || `${draftDataDir}/files`, 56)}</p>

      <button
        type="button"
        className="vscode-btn vscode-btn-primary flex w-full items-center justify-center gap-1"
        onClick={onSave}
        disabled={saving || !draftDataDir.trim()}
      >
        <Save className="h-3.5 w-3.5" />
        {saving ? "A guardar…" : "Aplicar pastas"}
      </button>
      {settings?.mode && (
        <p className="mt-2 text-[10px] text-vscode-muted">
          Modo {settings.mode}
          {settings.version ? ` · v${settings.version}` : ""}
        </p>
      )}
    </CollapsibleSection>
  );
}
