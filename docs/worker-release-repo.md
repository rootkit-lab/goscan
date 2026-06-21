# Deploy remoto via repositório privado

Para instalar `goscan-remote` em **muitos VPS** sem enviar o binário por SFTP a partir da tua máquina, publica releases num repositório git **privado** e deixa cada filho fazer `git clone`/`fetch`.

## Fluxo

```
Workstation                         Repositório privado              VPS (×N)
───────────                         ───────────────────              ────────
make publish-worker  ──push──►  goscan-worker-releases.git
                                         │
Settings → URL do repo                   │
Scan → Deploy antes                      │
     └─ SSH: git fetch + install ◄───────┘
```

## 1. Criar repositório privado

GitHub, GitLab ou servidor git próprio — **vazio**, só para binários:

```bash
# Exemplo GitHub (repo privado goscan-worker-releases)
```

## 2. Publicar release

```bash
export WORKER_RELEASE_REMOTE=git@github.com:SEU_USER/goscan-worker-releases.git

# primeira vez (clona o repo vazio)
make init-worker-release-repo

# build + commit + tag v{VERSION} + push
make publish-worker
```

Publica em `dist/worker-release/`:

| Ficheiro | Descrição |
|----------|-----------|
| `linux-amd64/goscan-remote` | Binário stateless (~7 MB) |
| `linux-amd64/VERSION` | Versão |
| `linux-amd64/BINHASH` | SHA256 para skip de deploy |
| `install-worker.sh` | Instalação manual no VPS |

Tag git: `v1.0.0` (lida de `assets/VERSION`).

## 3. Acesso nos VPS

### SSH (recomendado)

1. Gera deploy key read-only no GitHub/GitLab
2. Adiciona a chave **pública** em cada VPS (`~/.ssh/authorized_keys` ou como deploy key do repo)
3. URL na UI: `git@github.com:USER/goscan-worker-releases.git`

Cada VPS precisa de `git` instalado (`apt install git`).

### HTTPS + token

1. Token read-only (GitHub fine-grained ou classic)
2. Settings → **Deploy remoto** → URL `https://github.com/USER/goscan-worker-releases.git` + token

O token fica em `settings.yaml` (`0600`); nunca aparece nos logs.

## 4. Configurar GoScan

**Settings → Deploy remoto (workers)**

- **URL do repo** — clone SSH ou HTTPS
- **Ref** — vazio = tag `v{versão local}`; ou `main`, `v1.0.0`, etc.
- **Método** — `git` (default se URL preenchida) ou `sftp` (upload directo da workstation)

**Scan → Deploy/update remoto** — activo (comportamento actual).

Log típico com git:

```
[vps001] binário remoto desactualizado (sem BINHASH) — a actualizar…
[vps001] a actualizar via git (goscan-worker-releases · v1.0.0)…
[vps001] binário pronto no filho
```

Se hash OK: `deploy ignorado (v1.0.0 · hash OK)` — sem download.

## 5. Instalação manual (opcional)

Num VPS, sem orchestrador:

```bash
git clone --depth 1 --branch v1.0.0 git@github.com:USER/goscan-worker-releases.git
cd goscan-worker-releases
./install-worker.sh
```

## Variáveis

| Variável | Default | Uso |
|----------|---------|-----|
| `WORKER_RELEASE_REMOTE` | — | URL git do repo privado |
| `WORKER_RELEASE_DIR` | `dist/worker-release` | Clone local para publish |

## Segurança

- Repo **só binários** — não incluir código fonte nem credenciais
- Deploy keys **read-only** por VPS ou uma key partilhada com acesso só a este repo
- Preferir SSH a token HTTPS em produção

## Fallback SFTP

Se o repo não estiver configurado, o deploy continua por SFTP (upload a partir da workstation), como antes.
