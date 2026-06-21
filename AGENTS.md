# AGENTS — goscan

Índice para agentes de IA. Detalhe em **docs/**.

## Layout

| Path | Conteúdo |
|------|----------|
| `cmd/goscan/` | CLI scan + `findings list` |
| `internal/scanner/` | pipeline HTTP .env |
| `internal/store/` | `domains` + `findings` (FTS5) |
| `internal/scripts/` | runner Python |
| `src/goscan-ui/` | Wails3 + React |
| `scripts/` | checkers Python + migração |
| `var/findings/` | armazém .env (gitignored) |

## Dados sensíveis

Nunca versionar: `var/`, `dominios.db`, `**/*.env`, `files/`.

## Tasks

Backlog local: [`.tasks/README.md`](.tasks/README.md). Skill: `.cursor/skills/goscan-branch-task/`.

## Comandos

```bash
make build
make scan
make findings-list QUERY=shampora
make migrate-findings
make dev-ui
```
