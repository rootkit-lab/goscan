# Test all envs

Correr todos os checkers compatíveis por finding, gravar resultados e logs completos.

```bash
make scripts-venv
make test-all-envs                           # todos os findings
make test-all-envs ARGS="--unopened-only"    # só nunca abertos
make test-all-envs ARGS="--finding-id 42"    # um finding
make test-all-envs ARGS="--script chk-smtp"  # um checker
make test-all-envs ARGS="--filter mysql"       # alias → chk-mysql
make test-all-envs ARGS="--filter db"          # mysql + postgres + mongodb
make test-all-envs ARGS="--quick"            # sem email/DB pesado
make test-all-envs ARGS="--limit 10"         # primeiros N findings
make test-all-envs ARGS="--threads 4"        # paralelo
make test-all-envs ARGS="--no-log"           # sem gravar logs
```

## Logs batch

Cada run grava em `var/logs/batch/{run_id}/`:

- `manifest.json` — totais e opções
- `summary.txt` — linhas do output
- `results.jsonl` — 1 JSON por check (status, errorClass, logPath)
- `by-finding/{domain}/chk-*.log` — stdout+stderr completo (passwords redactadas)
- `failures/smtp-top-errors.txt` e `db-top-errors.txt`
- symlink `var/logs/batch/latest`

## Análise

```bash
make batch-analyze
make batch-analyze ARGS="--last"
./bin/goscan batch-analyze var/logs/batch/20260620_211345
```

## Email batch

SMTP / SendGrid / Mailgun em modo batch enviam para **rootmasters@proton.me** com assunto e mensagem aleatórios.

## UI

Painel **Scripts** → **Test all envs**, ou Ctrl+K. Após o batch, botão **Logs** no painel Output abre a pasta do run.
