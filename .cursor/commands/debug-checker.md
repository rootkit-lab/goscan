# Debug checker

Testar ou corrigir **um** checker num `.env` concreto.

## Recolher contexto

- `script_id` (ex. `chk-smtp`) — registry em `scripts/registry.yaml`
- Caminho do `.env` — finding em `var/findings/by-domain/…` ou path que o utilizador der
- Modo: interactivo (PTY/UI) vs batch (`--batch`)

## Comandos

```bash
cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
make scripts-venv

# Batch (sem prompts, grava SUMMARY)
GOSCAN_BATCH=1 scripts/.venv/bin/python -u scripts/chk-smtp.py \
  --env /caminho/para/.env --batch

# Interactivo no terminal
scripts/.venv/bin/python -u scripts/chk-smtp.py --env /caminho/para/.env

# Via CLI batch (todos compatíveis desse finding)
make test-all-envs ARGS="--finding-id ID --script chk-smtp"
```

## Ao debugar hang / sem output

1. Confirmar `PYTHONUNBUFFERED` / `-u` e `flush=True` nos prints
2. Confirmar `run_with_timeout` em operações de rede
3. SMTP: não usar `resolve_host` no MAIL_HOST
4. DB: confirmar `resolve_host` substitui `127.0.0.1` pelo domínio do finding

## Entrega

- Correr o checker (não só sugerir)
- Mostrar output ou erro real
- Fix mínimo se estiver claro o bug
- Resumo em português
