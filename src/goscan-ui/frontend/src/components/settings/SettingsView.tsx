import { Bell, FolderOpen, Save, Terminal } from "lucide-react";
import { ViewShell } from "@/components/layout/ViewShell";
import { RemoteWorkersSection, type DraftWorker } from "@/components/settings/RemoteWorkersSection";
import { DeployRepoSection } from "@/components/settings/DeployRepoSection";
import type { SettingsDTO } from "@/lib/api";

type Props = {
  settings: SettingsDTO | null;
  draftDataDir: string;
  draftScanDir: string;
  draftPythonPath: string;
  draftNotifyEnvFound: boolean;
  draftNotifyScriptOk: boolean;
  draftSoundEnvFound: boolean;
  draftSoundScriptOk: boolean;
  draftHubEnabled: boolean;
  draftWorkers: DraftWorker[];
  draftDeployRepoUrl: string;
  draftDeployRepoRef: string;
  draftDeployRepoToken: string;
  draftDeployRepoMethod: string;
  deployRepoHasToken: boolean;
  onDraftWorkersChange: (workers: DraftWorker[]) => void;
  onDeployRepoChange: (patch: { url?: string; ref?: string; method?: string; token?: string }) => void;
  onDraftHubEnabledChange: (v: boolean) => void;
  onPickKey: (index: number) => void;
  onTestWorker: (index: number) => void;
  onDraftDataDirChange: (v: string) => void;
  onDraftScanDirChange: (v: string) => void;
  onDraftPythonPathChange: (v: string) => void;
  onDraftNotifyEnvFoundChange: (v: boolean) => void;
  onDraftNotifyScriptOkChange: (v: boolean) => void;
  onDraftSoundEnvFoundChange: (v: boolean) => void;
  onDraftSoundScriptOkChange: (v: boolean) => void;
  onPickDataDir: () => void;
  onPickScanDir: () => void;
  onPickPython: () => void;
  onSave: () => void;
  onOpenDataDir: () => void;
  onOpenScanDir: () => void;
  saving?: boolean;
};

function shortPath(p: string, max = 48) {
  if (!p) return "—";
  if (p.length <= max) return p;
  return "…" + p.slice(-(max - 1));
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-gs-border bg-gs-surface p-4">
      <h3 className="mb-3 text-[11px] font-semibold uppercase tracking-wide text-gs-zone-header">{title}</h3>
      {children}
    </section>
  );
}

function ToggleRow({
  label,
  description,
  checked,
  onChange,
  disabled
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <label className="flex cursor-pointer items-start gap-3 rounded-md border border-transparent px-1 py-2 hover:border-gs-border/60 hover:bg-gs-hover/30">
      <input
        type="checkbox"
        className="mt-0.5 accent-[var(--gs-accent)]"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="min-w-0">
        <span className="block text-[12px] font-medium text-gs-fg">{label}</span>
        <span className="block text-[10px] text-gs-muted">{description}</span>
      </span>
    </label>
  );
}

export function SettingsView({
  settings,
  draftDataDir,
  draftScanDir,
  draftPythonPath,
  draftNotifyEnvFound,
  draftNotifyScriptOk,
  draftSoundEnvFound,
  draftSoundScriptOk,
  draftHubEnabled,
  draftWorkers,
  draftDeployRepoUrl,
  draftDeployRepoRef,
  draftDeployRepoToken,
  draftDeployRepoMethod,
  deployRepoHasToken,
  onDraftWorkersChange,
  onDeployRepoChange,
  onDraftHubEnabledChange,
  onPickKey,
  onTestWorker,
  onDraftDataDirChange,
  onDraftScanDirChange,
  onDraftPythonPathChange,
  onDraftNotifyEnvFoundChange,
  onDraftNotifyScriptOkChange,
  onDraftSoundEnvFoundChange,
  onDraftSoundScriptOkChange,
  onPickDataDir,
  onPickScanDir,
  onPickPython,
  onSave,
  onOpenDataDir,
  onOpenScanDir,
  saving
}: Props) {
  return (
    <ViewShell title="Configuração">
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-4 pb-6">
        {settings?.needsSetup && (
          <p className="rounded border border-amber-700/50 bg-amber-950/40 px-3 py-2 text-[11px] text-amber-100">
            Configure pastas de produção (separadas do repo de dev). Corra{" "}
            <code className="text-amber-50">make migrate-prod-data</code> para copiar dados existentes.
          </p>
        )}
        {settings?.pointsToDevRepo && (
          <p className="rounded border border-red-800/50 bg-red-950/40 px-3 py-2 text-[11px] text-red-100">
            Atenção: prod aponta para o repo de dev. Use{" "}
            <code className="text-red-50">{settings.defaultProdDataDir || "~/.local/share/goscan/data"}</code>.
          </p>
        )}

        <Section title="Pastas">
          <label className="mb-1 block text-[11px] text-gs-muted">Dados (DB, findings, logs)</label>
          <div className="mb-1 flex gap-1">
            <input
              className="gs-input min-w-0 flex-1 text-[11px]"
              value={draftDataDir}
              onChange={(e) => onDraftDataDirChange(e.target.value)}
              placeholder={settings?.defaultProdDataDir || "/caminho/para/dados"}
            />
            <button type="button" className="gs-btn shrink-0 px-2" onClick={onPickDataDir}>
              …
            </button>
            <button type="button" className="gs-btn shrink-0 px-2" onClick={onOpenDataDir}>
              <FolderOpen className="h-3.5 w-3.5" />
            </button>
          </div>
          <p className="mb-4 text-[10px] text-gs-muted">{shortPath(draftDataDir)}</p>

          <label className="mb-1 block text-[11px] text-gs-muted">Domínios para scan (.txt, .env)</label>
          <div className="mb-1 flex gap-1">
            <input
              className="gs-input min-w-0 flex-1 text-[11px]"
              value={draftScanDir}
              onChange={(e) => onDraftScanDirChange(e.target.value)}
              placeholder="(default: dados/files)"
            />
            <button type="button" className="gs-btn shrink-0 px-2" onClick={onPickScanDir}>
              …
            </button>
            <button type="button" className="gs-btn shrink-0 px-2" onClick={onOpenScanDir}>
              <FolderOpen className="h-3.5 w-3.5" />
            </button>
          </div>
          <p className="text-[10px] text-gs-muted">{shortPath(draftScanDir || `${draftDataDir}/files`)}</p>
        </Section>

        <Section title="Sistema">
          <label className="mb-1 block text-[11px] text-gs-muted">Python dos checkers</label>
          <div className="mb-1 flex gap-1">
            <input
              className="gs-input min-w-0 flex-1 font-mono text-[11px]"
              value={draftPythonPath}
              onChange={(e) => onDraftPythonPathChange(e.target.value)}
              placeholder="(auto: venv do projecto ou python3)"
            />
            <button type="button" className="gs-btn shrink-0 px-2" onClick={onPickPython}>
              …
            </button>
          </div>
          <p className="flex items-center gap-1.5 text-[10px] text-gs-muted">
            <Terminal className="h-3 w-3 shrink-0" />
            Em uso:{" "}
            <span className="truncate font-mono text-gs-fg">
              {settings?.pythonPathEffective || "python3"}
            </span>
          </p>
        </Section>

        <Section title="Alertas">
          <div className="mb-1 flex items-center gap-2 text-[11px] text-gs-muted">
            <Bell className="h-3.5 w-3.5" />
            Notificações e sons durante scan e checkers
          </div>
          <div className="grid gap-1 sm:grid-cols-2">
            <ToggleRow
              label="Notificar .env encontrado"
              description="Alerta de sistema quando o scan encontra um novo finding."
              checked={draftNotifyEnvFound}
              onChange={onDraftNotifyEnvFoundChange}
            />
            <ToggleRow
              label="Som .env encontrado"
              description="Bip curto ao detectar um .env."
              checked={draftSoundEnvFound}
              onChange={onDraftSoundEnvFoundChange}
            />
            <ToggleRow
              label="Notificar checker OK"
              description="Alerta quando um script termina com sucesso."
              checked={draftNotifyScriptOk}
              onChange={onDraftNotifyScriptOkChange}
            />
            <ToggleRow
              label="Som checker OK"
              description="Bip ao confirmar credenciais válidas."
              checked={draftSoundScriptOk}
              onChange={onDraftSoundScriptOkChange}
            />
          </div>
        </Section>

        <DeployRepoSection
          url={draftDeployRepoUrl}
          repoRef={draftDeployRepoRef}
          method={draftDeployRepoMethod}
          token={draftDeployRepoToken}
          hasToken={deployRepoHasToken}
          disabled={saving}
          onChange={onDeployRepoChange}
        />

        <Section title="Hub remoto (stream em tempo real)">
          <label className="flex cursor-pointer items-start gap-3 rounded-md border border-gs-border/60 px-2 py-2 hover:border-gs-border">
            <input
              type="checkbox"
              className="mt-0.5 accent-[var(--gs-accent)]"
              checked={draftHubEnabled}
              disabled={saving}
              onChange={(e) => onDraftHubEnabledChange(e.target.checked)}
            />
            <span className="min-w-0">
              <span className="block text-[12px] font-medium text-gs-fg">Activar hub (socket)</span>
              <span className="block text-[10px] text-gs-muted">
                Recebe progresso e conteúdo .env em tempo real via túnel SSH. Desactive para usar apenas stderr/export.
              </span>
            </span>
          </label>
        </Section>

        <RemoteWorkersSection
          workers={draftWorkers}
          onChange={onDraftWorkersChange}
          onPickKey={onPickKey}
          onTest={onTestWorker}
          disabled={saving}
        />

        <button
          type="button"
          className="gs-btn gs-btn-primary flex w-full items-center justify-center gap-1"
          onClick={onSave}
          disabled={saving || !draftDataDir.trim()}
        >
          <Save className="h-3.5 w-3.5" />
          {saving ? "A guardar…" : "Guardar"}
        </button>

        {settings?.mode && (
          <p className="text-center text-[10px] text-gs-muted">
            Modo {settings.mode}
            {settings.version ? ` · v${settings.version}` : ""}
          </p>
        )}
      </div>
    </ViewShell>
  );
}
