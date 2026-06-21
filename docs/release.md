# Releases (GitHub Actions)

## Artefactos

| Plataforma | Ficheiro | Conteúdo |
|------------|----------|----------|
| Linux amd64 | `goscan_VERSION_amd64.deb` | CLI + UI + goscan-remote + scripts + venv (postinst) |
| Windows amd64 | `goscan_VERSION_amd64.msi` | CLI + UI + scripts (Program Files\GoScan) |

## Fluxo integrado (`/criar-release`)

### Instalação local (dev → prod na máquina)

```bash
make scripts-venv
cd src/goscan-ui/frontend && npm install   # se necessário
make release VERSION=1.0.1                 # opcional: bump
make install
make install-doctor
```

### Publicar no GitHub (instaladores + Release)

```bash
make release-publish VERSION=1.0.1
```

Isto:

1. Grava `assets/VERSION`
2. Commit + tag `v1.0.1`
3. Push `main` + tag
4. Dispara [`.github/workflows/release.yml`](../.github/workflows/release.yml):
   - job **Linux** → `.deb`
   - job **Windows** → `.msi`
   - job **github-release** → anexa ambos à Release

Acompanhar: https://github.com/rootkit-lab/goscan/actions

### Manual (sem push)

```bash
make package-deb    # dist/goscan_*_amd64.deb (Linux)
make package-msi    # dist/goscan_*_amd64.msi (Windows + WiX)
```

## Instalar o .deb (Ubuntu/Debian)

```bash
sudo dpkg -i dist/goscan_1.0.0_amd64.deb
sudo apt-get install -f   # se faltar dependência
goscan-ui
```

Dados do utilizador: `~/.local/share/goscan/data/` (settings em `~/.config/goscan/settings.yaml`).

## Requisitos CI

- **Linux:** `python3`, `python3-venv`, `dpkg-deb`, Node 20, Go 1.25
- **Windows:** WiX 5 (`dotnet tool install --global wix`), Git Bash, Node 20, Go 1.25

## Dev vs prod vs release

| | Dev (`make dev-ui`) | Prod local (`make install`) | `.deb` / `.msi` |
|--|---------------------|----------------------------|-----------------|
| App | repo | `~/.local/share/goscan/app` | `/usr/local/share/goscan/app` ou `Program Files\GoScan` |
| Dados | repo `var/` | `~/.local/share/goscan/data` | idem (por utilizador) |
