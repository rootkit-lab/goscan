# Novo checker

Criar um checker Python compatível com batch, UI e registry.

## Input esperado

O utilizador indica o serviço (ex.: MongoDB, Slack webhook, Elasticsearch). Se faltar, inferir do pedido.

## Checklist de implementação

1. **`scripts/chk-<nome>.py`**
   - Shebang + docstring
   - `env_arg_parser` + `--batch`
   - `is_batch_mode` → `run_batch()` sem prompts
   - `is_interactive()` → prompts PTY
   - Fallback não-interactivo com `print_summary("…")`
   - `log_step`, `run_with_timeout`, `format_network_error`
   - DB/API: timeout 15–25s; hosts locais via `ctx.resolve_host()` (excepto SMTP — host raw)

2. **`scripts/registry.yaml`** — `id`, `path`, `label`, `interactive: true`, `env_keys`

3. **`scripts/requirements.txt`** — dependências novas

4. **`src/goscan-ui/frontend/src/lib/scriptIcons.ts`** — ícone lucide

5. **`internal/scripts/batch.go`** — se for email (`emailScripts`) ou DB pesado (`heavyScripts`)

6. Verificar: `scripts/.venv/bin/python scripts/chk-….py --env … --batch`

## Email (se aplicável)

- Batch: `GOSCAN_TEST_EMAIL` + `random_email_content()` → `rootmasters@proton.me`

## Não fazer

- Não commitar `.env` de teste
- Não criar docs extra além do registry

No fim: `/verify` e exemplo de comando para testar manualmente.
