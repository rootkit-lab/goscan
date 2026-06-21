---
related_skill: goscan-branch-task
---

# Criar release

Compila GoScan em modo **produção** e **instala/atualiza** no sistema (`~/.local`), sem misturar dados com o repo de desenvolvimento.

## Pré-requisitos

```bash
make scripts-venv    # uma vez
cd src/goscan-ui/frontend && npm install   # se frontend nunca buildou
```

## Fluxo (agente)

1. Ler `.tasks/feat/system-install-release.md` — checklist activa
2. Confirmar que **`make dev-ui` não precisa de ser parado** (dev usa repo + porta 9280; prod usa `~/.local/share/goscan/data`)
3. Opcional: actualizar `assets/VERSION`
4. **`make release`** — compila `bin/goscan` + `bin/goscan-ui` + frontend
5. **`make install`** — copia para `~/.local`, venv, symlinks, `.desktop`, ícone
6. **`make install-doctor`** — validar mode/paths/versão
7. Teste rápido: `~/.local/bin/goscan findings list --limit 3`
8. Marcar checkboxes da task concluídos

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
