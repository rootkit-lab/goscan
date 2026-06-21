---
related_skill: goscan-branch-task
---

# Criar release

Compila GoScan em modo **produção**, instala localmente ou **publica instaladores** (.deb + .msi) via GitHub Actions.

## Pré-requisitos

```bash
make scripts-venv    # uma vez
cd src/goscan-ui/frontend && npm install   # se frontend nunca buildou
```

## Fluxo local (agente)

1. Confirmar que **`make dev-ui` não precisa de ser parado** (dev usa repo; prod usa `~/.local/share/goscan/data`)
2. Opcional: **`make release VERSION=1.0.1`** — bump `assets/VERSION` antes do build
3. **`make release`** — compila `bin/goscan`, `bin/goscan-ui`, `bin/goscan-remote` + frontend
4. **`make install`** — copia para `~/.local`, venv, symlinks, `.desktop`, ícone
5. **`make install-doctor`** — validar mode/paths/versão
6. Teste rápido: `~/.local/bin/goscan findings list --limit 3`

## Fluxo GitHub Release (instaladores)

Publica tag `vX.Y.Z` e dispara CI que gera **`.deb`** (Linux) e **`.msi`** (Windows) na Release do GitHub.

```bash
make release-publish VERSION=1.0.1
```

Ou manualmente: actualizar `assets/VERSION`, commit, `git tag v1.0.1`, `git push origin main --tags`.

| Passo | Comando / resultado |
|-------|---------------------|
| Publicar | `make release-publish VERSION=1.0.1` |
| CI | `.github/workflows/release.yml` |
| Artefactos | `dist/goscan_1.0.1_amd64.deb`, `dist/goscan_1.0.1_amd64.msi` |
| Instalar deb | `sudo dpkg -i goscan_*.deb` |

Detalhe: [`docs/release.md`](../docs/release.md)

### Empacotar só localmente (sem push)

```bash
make package-deb    # Linux — requer dpkg-deb
make package-msi    # Windows — requer WiX (`dotnet tool install -g wix`)
```

## Dev vs prod

| | Desenvolvimento | Produção (instalado) |
|--|-----------------|----------------------|
| Comando | `make dev-ui` | `goscan-ui` (menu ou terminal) |
| App (scripts) | repo `scripts/` | `~/.local/share/goscan/app/` |
| Dados (DB, findings) | repo `dominios.db`, `var/` | `~/.local/share/goscan/data/` |
| Mode | `dev` (`GOSCAN_REPO_ROOT`) | `prod` (`GOSCAN_MODE=prod` no .desktop) |

**Nunca** correr `make install` esperando que actualize o ambiente do `make dev-ui` — são dados separados.

## Comandos

```bash
make release                  # build prod
make release VERSION=1.0.1    # bump assets/VERSION antes do build
make install                  # instala/atualiza ~/.local
make install PREFIX=$HOME/.local
make uninstall                # remove binários/desktop (mantém data/)
make install-doctor
make release-publish VERSION=1.0.1   # tag + GitHub Actions
make package-deb
make package-msi
```

## Desinstalar só app (manter findings prod)

```bash
make uninstall
# dados ficam em ~/.local/share/goscan/data/
```

## Se algo falhar

| Problema | Acção |
|----------|--------|
| UI instalada sem checkers | `make install` (recria venv em app/scripts/.venv) |
| Ícone em falta | `python3 scripts/icon-to-png.py` + `make install` |
| Prod a usar repo | `make install-doctor` — verificar `GOSCAN_MODE` |
| `python3-venv` em falta | `sudo apt install python3-venv python3-full` |
| CI MSI falha | Verificar job Windows — WiX 5 + Git Bash |
| CI deb falha | `sudo apt install python3-venv` no runner (já no workflow) |
