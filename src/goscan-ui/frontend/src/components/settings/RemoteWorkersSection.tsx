import { Plus, Trash2, Wifi } from "lucide-react";
import { clsx } from "clsx";
import type { RemoteWorkerSaveDTO, RemoteWorkerTestResultDTO } from "@/lib/api";

export type DraftWorker = RemoteWorkerSaveDTO & {
  testResult?: RemoteWorkerTestResultDTO;
  testing?: boolean;
};

type Props = {
  workers: DraftWorker[];
  onChange: (workers: DraftWorker[]) => void;
  onPickKey: (index: number) => void;
  onTest: (index: number) => void;
  disabled?: boolean;
};

function newWorker(): DraftWorker {
  const id = `worker-${Math.random().toString(16).slice(2, 10)}`;
  return {
    id,
    name: "",
    host: "",
    port: 22,
    user: "root",
    authType: "password",
    password: "",
    keyPath: "",
    keyPassphrase: "",
    execMode: "ssh",
    apiPort: 9090,
    apiToken: "",
    enabled: true
  };
}

export function RemoteWorkersSection({ workers, onChange, onPickKey, onTest, disabled }: Props) {
  const update = (index: number, patch: Partial<DraftWorker>) => {
    onChange(workers.map((w, i) => (i === index ? { ...w, ...patch, testResult: undefined } : w)));
  };

  const remove = (index: number) => {
    onChange(workers.filter((_, i) => i !== index));
  };

  return (
    <section className="rounded-lg border border-gs-border bg-gs-surface p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <h3 className="text-[11px] font-semibold uppercase tracking-wide text-gs-zone-header">Workers remotos</h3>
        <button
          type="button"
          className="gs-btn flex items-center gap-1 px-2 py-1 text-[11px]"
          disabled={disabled}
          onClick={() => onChange([...workers, newWorker()])}
        >
          <Plus className="h-3.5 w-3.5" />
          Adicionar
        </button>
      </div>

      {workers.length === 0 && (
        <p className="text-[11px] text-gs-muted">Nenhum worker configurado — scan só local.</p>
      )}

      <div className="flex flex-col gap-3">
        {workers.map((w, index) => (
          <div key={w.id || index} className="rounded-md border border-gs-border/70 bg-gs-bg/40 p-3">
            <div className="mb-2 grid gap-2 sm:grid-cols-2">
              <label className="block">
                <span className="mb-1 block text-[10px] text-gs-muted">Nome</span>
                <input
                  className="gs-input w-full text-[11px]"
                  value={w.name}
                  disabled={disabled}
                  onChange={(e) => update(index, { name: e.target.value })}
                  placeholder="VPS-EU-1"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-[10px] text-gs-muted">Host</span>
                <input
                  className="gs-input w-full text-[11px]"
                  value={w.host}
                  disabled={disabled}
                  onChange={(e) => update(index, { host: e.target.value })}
                  placeholder="203.0.113.10"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-[10px] text-gs-muted">Porta SSH</span>
                <input
                  type="number"
                  className="gs-input w-full text-[11px]"
                  value={w.port || 22}
                  disabled={disabled}
                  onChange={(e) => update(index, { port: Number(e.target.value) || 22 })}
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-[10px] text-gs-muted">Utilizador</span>
                <input
                  className="gs-input w-full text-[11px]"
                  value={w.user}
                  disabled={disabled}
                  onChange={(e) => update(index, { user: e.target.value })}
                />
              </label>
            </div>

            <div className="mb-2 flex flex-wrap gap-3 text-[11px]">
              <span className="text-gs-muted">Modo:</span>
              {(["ssh", "http"] as const).map((mode) => (
                <label key={mode} className="flex items-center gap-1">
                  <input
                    type="radio"
                    name={`exec-${w.id}`}
                    checked={w.execMode === mode}
                    disabled={disabled}
                    onChange={() => update(index, { execMode: mode })}
                  />
                  {mode === "ssh" ? "SSH directo" : "HTTP API"}
                </label>
              ))}
            </div>

            <div className="mb-2 flex flex-wrap gap-3 text-[11px]">
              <span className="text-gs-muted">Auth:</span>
              {(["password", "key", "ppk"] as const).map((auth) => (
                <label key={auth} className="flex items-center gap-1">
                  <input
                    type="radio"
                    name={`auth-${w.id}`}
                    checked={w.authType === auth}
                    disabled={disabled}
                    onChange={() => update(index, { authType: auth })}
                  />
                  {auth === "password" ? "Password" : auth === "key" ? "Chave PEM" : "PPK"}
                </label>
              ))}
            </div>

            {w.authType === "password" ? (
              <label className="mb-2 block">
                <span className="mb-1 block text-[10px] text-gs-muted">Password</span>
                <input
                  type="password"
                  className="gs-input w-full text-[11px]"
                  value={w.password}
                  disabled={disabled}
                  onChange={(e) => update(index, { password: e.target.value })}
                  placeholder="••••••••"
                />
              </label>
            ) : (
              <div className="mb-2 flex gap-1">
                <input
                  className="gs-input min-w-0 flex-1 text-[11px]"
                  value={w.keyPath}
                  disabled={disabled}
                  onChange={(e) => update(index, { keyPath: e.target.value })}
                  placeholder="/home/user/.ssh/id_rsa"
                />
                <button
                  type="button"
                  className="gs-btn shrink-0 px-2 text-[11px]"
                  disabled={disabled}
                  onClick={() => onPickKey(index)}
                >
                  …
                </button>
              </div>
            )}

            {w.execMode === "http" && (
              <label className="mb-2 block">
                <span className="mb-1 block text-[10px] text-gs-muted">Porta API</span>
                <input
                  type="number"
                  className="gs-input w-32 text-[11px]"
                  value={w.apiPort || 9090}
                  disabled={disabled}
                  onChange={(e) => update(index, { apiPort: Number(e.target.value) || 9090 })}
                />
              </label>
            )}

            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                className="gs-btn flex items-center gap-1 px-2 py-1 text-[11px]"
                disabled={disabled || w.testing || !w.host.trim()}
                onClick={() => onTest(index)}
              >
                <Wifi className="h-3.5 w-3.5" />
                {w.testing ? "A testar…" : "Testar ligação"}
              </button>
              <button
                type="button"
                className="gs-btn flex items-center gap-1 px-2 py-1 text-[11px] text-gs-error"
                disabled={disabled}
                onClick={() => remove(index)}
              >
                <Trash2 className="h-3.5 w-3.5" />
                Remover
              </button>
              {w.testResult && (
                <span
                  className={clsx(
                    "text-[10px]",
                    w.testResult.ok ? "text-gs-success" : "text-gs-error"
                  )}
                >
                  {w.testResult.ok
                    ? `● OK${w.testResult.remoteVersion ? ` · goscan ${w.testResult.remoteVersion}` : ""}`
                    : w.testResult.error ?? "Falhou"}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
