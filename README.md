# goscan

Scanner de ficheiros `.env` expostos via HTTP, com UI desktop (Wails + React), checkers Python de credenciais e scan orquestrado em workers locais/remotos.

> **Uso responsável:** utilize apenas em sistemas e domínios sobre os quais tem autorização explícita. Os findings (conteúdo `.env`) são dados sensíveis e ficam **sempre locais** — nunca são versionados neste repositório.

## Funcionalidades

- Scan HTTP de paths `.env` comuns (modo normal ou `-fast`)
- Base SQLite de domínios + findings com pesquisa FTS5
- UI estilo workbench: editor Monaco, terminal, painel de scan, settings
- Scripts `chk-*.py` para validar credenciais encontradas (`--env PATH`)
- Scan orquestrado: fila central, workers SSH remotos, hub WebSocket encriptado
- Alertas nativos Linux ao encontrar `.env` (`notify-send`)

## Requisitos

- Go 1.22+
- Python 3 + `python3-venv` (checkers)
- Node.js + npm (UI)
- [Wails v3](https://wails.io/) (`wails3` no PATH) — apenas para `make dev-ui` / release UI
- Linux (Pop!_OS / Ubuntu testados; alertas desktop via `libnotify-bin`)

## Instalação rápida (dev)

```bash
git clone https://github.com/rootkit-lab/goscan.git
cd goscan

make scripts-venv
make build
make setup-hooks   # opcional: pre-commit bloqueia dados sensíveis
```

### UI em desenvolvimento

```bash
make dev-ui
```

### Instalação em produção (XDG)

```bash
make release && make install
# → ~/.local/share/goscan/app/
# → settings em ~/.config/goscan/settings.yaml
```

## Comandos úteis

| Comando | Descrição |
|---------|-----------|
| `make scan` | Scan CLI em `files/` |
| `make findings-list QUERY=exemplo` | Pesquisar findings |
| `make test` | Testes Go |
| `make test-checkers-smoke` | Smoke dos checkers Python |
| `make install-doctor` | Comparar dev vs prod |

Documentação adicional em [`docs/`](docs/) e [`AGENTS.md`](AGENTS.md).

## Layout

```
cmd/goscan/          CLI principal (+ scan orquestrado)
cmd/goscan-remote/   Worker remoto stateless (sem SQLite)
internal/scanner/    Pipeline HTTP
internal/store/      Domínios + findings (FTS5)
internal/scanorch/   Orquestrador multi-worker
src/goscan-ui/       Desktop UI (Wails3 + React)
scripts/             Checkers Python + registry.yaml
```

## Dados locais (não versionados)

| Path | Conteúdo |
|------|----------|
| `var/findings/` | Ficheiros `.env` encontrados |
| `dominios.db` | SQLite de domínios/findings |
| `files/` | Listas de entrada (ex.: domínios) |
| `~/.config/goscan/settings.yaml` | Workers SSH, tokens deploy, preferências |
| `config.yml` | Overrides locais (copiar de `config.yml.example`) |

## Workers remotos

1. Publicar `goscan-remote` num repositório **privado** de releases:

   ```bash
   export WORKER_RELEASE_REMOTE=git@github.com:rootkit-lab/goscan-worker-releases.git
   make publish-worker
   ```

2. Configurar workers na UI (Settings → Workers remotos).

Ver [`docs/remote-scan.md`](docs/remote-scan.md) e [`docs/worker-release-repo.md`](docs/worker-release-repo.md).

## Releases (.deb / .msi)

Instaladores gerados pelo GitHub Actions em cada tag `v*`:

```bash
make release-publish VERSION=1.0.1   # tag + push → CI
# ou: Actions → Release → Run workflow
```

Ver [`docs/release.md`](docs/release.md).

## Licença

MIT — ver [LICENSE](LICENSE).
