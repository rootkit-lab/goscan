# Dev UI

Porta **9280** (fora do stack RemoteWindows: 9245–9251, 5173).

`make dev-ui` cria/atualiza o venv Python em `scripts/.venv` (checkers interactivos) — **não** uses `pip install` global.

```bash
make dev-ui
make dev-ui GOSCAN_UI_PORT=9282   # se a porta estiver ocupada
```

Processo antigo: feche a janela goscan ou `pkill -f 'goscan-ui/frontend.*vite'`.
