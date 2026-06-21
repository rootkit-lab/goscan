# Dev — ajuda rápida

Lista os comandos Cursor deste projecto e sugere o próximo passo consoante o pedido do utilizador.

## Comandos disponíveis

| Comando | Quando usar |
|---------|-------------|
| `/dev-setup` | Primeira vez ou ambiente partido (venv, build, hooks) |
| `/dev-ui` | Arrancar UI Wails (porta 9280) |
| `/verify` | Antes de commit/PR — testes + build Go + frontend + Python |
| `/status` | Snapshot: git, task activa, portas, findings |
| `/criar-task` | Nova feature — checklist em `.tasks/` |
| `/implementar-task` | Executar task activa do backlog |
| `/novo-checker` | Adicionar script Python + registry + ícone UI |
| `/debug-checker` | Correr um checker num `.env` (manual ou batch) |
| `/test-all-envs` | Batch de checkers (CLI) |
| `/run-scan` | Scan de domínios |

## Mapa do repo

| Área | Path |
|------|------|
| CLI | `cmd/goscan/` |
| Batch checkers | `internal/scripts/batch*.go`, `cmd/goscan/testall.go` |
| Store / findings | `internal/store/` |
| UI Wails | `src/goscan-ui/` |
| Frontend | `src/goscan-ui/frontend/` |
| Checkers Python | `scripts/chk-*.py`, `scripts/envutil.py`, `scripts/registry.yaml` |
| Tasks locais | `.tasks/` (não versionado) |

## Regras rápidas

- **Nunca commitar** `var/`, `*.env`, credenciais
- Checkers: `--env PATH`, modo batch `--batch` ou `GOSCAN_BATCH=1`
- Email batch: destino fixo `rootmasters@proton.me` (ver `envutil.py`)
- UI: `make dev-ui` (não `pip` global — usa `scripts/.venv`)

Responde em português. Se o pedido encaixar num comando acima, recomenda-o explicitamente.
