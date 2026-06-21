import { GitBranch } from "lucide-react";

type Props = {
  url: string;
	repoRef: string;
  method: string;
  token: string;
  hasToken: boolean;
  onChange: (patch: { url?: string; ref?: string; method?: string; token?: string }) => void;
  disabled?: boolean;
};

export function DeployRepoSection({ url, repoRef, method, token, hasToken, onChange, disabled }: Props) {
  const gitMode = method !== "sftp" && url.trim() !== "";

  return (
    <section className="rounded-lg border border-gs-border bg-gs-surface p-4">
      <h3 className="mb-1 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-wide text-gs-zone-header">
        <GitBranch className="h-3.5 w-3.5" />
        Deploy remoto (repositório)
      </h3>
      <p className="mb-3 text-[10px] text-gs-muted">
        Repositório git privado com <code className="text-gs-fg">goscan-remote</code>. Cada VPS faz{" "}
        <code className="text-gs-fg">git fetch</code> em vez de SFTP — ideal para muitos servidores. Ver{" "}
        <code className="text-gs-fg">docs/worker-release-repo.md</code>.
      </p>

      <div className="grid gap-2">
        <label className="block text-[11px] text-gs-muted">
          URL do repo
          <input
            type="text"
            className="gs-input mt-0.5 w-full font-mono text-[11px]"
            placeholder="git@github.com:USER/goscan-worker-releases.git"
            value={url}
            disabled={disabled}
            onChange={(e) => onChange({ url: e.target.value })}
          />
        </label>

        <div className="grid gap-2 sm:grid-cols-2">
          <label className="block text-[11px] text-gs-muted">
            Ref (tag/branch)
            <input
              type="text"
              className="gs-input mt-0.5 w-full font-mono text-[11px]"
              placeholder="v1.0.0 (vazio = versão local)"
				value={repoRef}
              disabled={disabled}
              onChange={(e) => onChange({ ref: e.target.value })}
            />
          </label>

          <label className="block text-[11px] text-gs-muted">
            Método
            <select
              className="gs-input mt-0.5 w-full text-[11px]"
              value={method || (url.trim() ? "git" : "sftp")}
              disabled={disabled}
              onChange={(e) => onChange({ method: e.target.value })}
            >
              <option value="git">git (repo privado)</option>
              <option value="sftp">sftp (upload desta máquina)</option>
            </select>
          </label>
        </div>

        {(url.startsWith("https://") || hasToken) && (
          <label className="block text-[11px] text-gs-muted">
            Token HTTPS {hasToken && !token ? "(guardado)" : ""}
            <input
              type="password"
              className="gs-input mt-0.5 w-full font-mono text-[11px]"
              placeholder={hasToken ? "•••••••• (deixar vazio para manter)" : "ghp_… ou glpat-…"}
              value={token}
              disabled={disabled}
              onChange={(e) => onChange({ token: e.target.value })}
            />
          </label>
        )}

        {gitMode && (
          <p className="text-[10px] text-gs-success">
            Deploy via git activo — publicar com <code className="text-gs-fg">make publish-worker</code>
          </p>
        )}
      </div>
    </section>
  );
}
