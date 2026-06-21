# Batch analyze

Analisar falhas do último «Test all envs» (SMTP/DB) a partir dos logs persistidos.

```bash
make batch-analyze
make batch-analyze ARGS="--last"
./bin/goscan batch-analyze var/logs/batch/20260620_211345
```

Lê `results.jsonl` + `manifest.json` e imprime top erros e sugestões.

Logs em `var/logs/batch/latest/` (symlink).
