# Dev UI

Arrancar ou diagnosticar a UI Wails (estilo VS Code).

## Arrancar

```bash
cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
make dev-ui
# porta alternativa:
make dev-ui GOSCAN_UI_PORT=9282
```

- Porta default **9280** (evitar 9245–9251, 5173)
- `make dev-ui` garante `scripts/.venv` + `GOSCAN_REPO_ROOT`
- **Não** uses `pip install` global nem `wails3 dev` sem o Makefile

## Se a porta estiver ocupada

```bash
pkill -f 'goscan-ui/frontend.*vite'   # fechar instância anterior
# ou outra porta: make dev-ui GOSCAN_UI_PORT=9282
```

## Checkers na UI

- **Run selected** — terminal PTY interactivo
- **Test all (finding/filtro/quick)** — batch no tab Output
- Ícones ✓/✗ na sidebar vêm de `checker_results`

## Se pedirem alteração UI

- Frontend: `src/goscan-ui/frontend/src/`
- Bindings: `src/goscan-ui/app.go`
- Depois: `cd src/goscan-ui/frontend && npm run build`

Corre `make dev-ui` em background se o utilizador quiser a UI aberta. Responde em português.
