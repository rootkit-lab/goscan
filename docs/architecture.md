# Arquitectura goscan

## Componentes

- **CLI** (`cmd/goscan`) — ingestão de domínios + scan HTTP `.env`
- **UI** (`src/goscan-ui`) — Wails3 + React: pesquisa FTS, Monaco, xterm, scripts, scan
- **Store** (`internal/store`) — `domains` + `findings` (FTS5) em `dominios.db`
- **Scripts** (`scripts/chk-*.py`) — validação de credenciais via `--env`

## Dados

```
var/findings/by-domain/{domain}/{label}.env
dominios.db  →  domains, findings, findings_fts
var/archive/scan_resultados_*  →  scans antigos
```

## Fluxo scan

1. Ingestão de `files/` (.txt / .env) → tabela `domains`
2. Scan HTTP paths → validação conteúdo → `findings` + ficheiro em `var/findings/`
3. UI/CLI pesquisam via FTS5

## Eventos Wails

| Evento | Payload |
|--------|---------|
| `script:stdout` | linha stdout |
| `script:stderr` | linha stderr |
| `script:exit` | `{ scriptId, exitCode }` |
| `scan:progress` | `ScanProgressDTO` |
| `scan:found` | `{ domain, url, path }` |
