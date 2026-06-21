# Dev setup

Preparar ambiente de desenvolvimento goscan do zero (ou reparar venv/build em falta).

## Executar (por ordem)

```bash
cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
make scripts-venv
make build
make test
cd src/goscan-ui/frontend && npm install && npm run build
make setup-hooks   # opcional, se existir .githooks
```

## Verificar

- `scripts/.venv/bin/python -c "import pymysql, redis, pymongo; print('OK')"`
- `./bin/goscan findings list --limit 3`
- `./bin/goscan test-all --help`

## Se falhar

| Erro | Acção |
|------|--------|
| `python3-venv` em falta | `sudo apt install python3-venv python3-full` |
| Porta 9280 ocupada | `make dev-ui GOSCAN_UI_PORT=9282` |
| Wails sem `GOSCAN_REPO_ROOT` | usar sempre `make dev-ui`, não `wails3 dev` directo |

## Produção (sistema)

```bash
make install              # ~/.local/bin + menu GoScan
make install-doctor       # confirma paths dev vs prod
```

- **Dev** (`make dev-ui`): dados em `dominios.db` + `var/` **no repo** (ou pasta escolhida em Pastas)
- **Prod** (`goscan-ui` instalado): escolher pastas no painel **Pastas** → grava em `~/.config/goscan/settings.yaml`
- Actualizar prod após mudanças: `/criar-release` ou `make release && make install`

Corre os comandos tu mesmo. No fim, resume o que ficou OK e o que falta.
