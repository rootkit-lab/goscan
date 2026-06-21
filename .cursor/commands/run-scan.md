# Run scan

Escanear domínios e gravar findings.

## CLI

```bash
cd /home/wiz/Projects/goscan
make scan
# ou com opções:
make build
./bin/goscan -dir files -threads 100 -fast -rescan
```

## UI

Painel **Actions → Scan → Start** (ou Ctrl+K → Start scan). Output no tab **Output**.

## Pós-scan

```bash
make findings-list
make migrate-findings   # se houver dumps antigos scan_resultados_*
```

## Dados

- Findings: `var/findings/by-domain/`
- SQLite: `var/dominios.db`

Corre o scan se o utilizador pedir execução. Resumo: domínios/vulns encontrados. Português.
