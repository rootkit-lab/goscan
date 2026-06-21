# Scan remoto — hub local + workers filhos

O **goscan local** é a fonte única de verdade. Workers remotos (filhos) só recebem batches de domínios, testam, e devolvem findings — nunca acumulam resultados permanentes.

## Modelo

```
┌─ GoScan LOCAL (master) ─────────────────────────────────────────┐
│  dominios.db  ·  var/findings/  ·  lista completa de domínios   │
│                                                                 │
│  1. Particiona domínios pendentes (cada um → 1 destino)         │
│  2. Envia batch único por worker (domains.txt via SFTP)         │
│  3. Recebe findings JSON e grava localmente                     │
│  4. Marca domínios como scaneados só após merge OK             │
└─────────────────────────────────────────────────────────────────┘
         │ batch A              │ batch B
         ▼                      ▼
   Worker filho            Worker filho
   (DB efémera /tmp)       (DB efémera /tmp)
   testa · devolve          testa · devolve
   · apaga temp            · apaga temp
```

## Garantias

| Regra | Como |
|-------|------|
| Domínios únicos por destino | Partição `rowid % N = índice` — disjoint, sem overlap |
| Filho só testa o batch | Recebe `domains.txt`; não lê pastas locais |
| Filho não guarda resultados | DB/findings em `/tmp/goscan-run-*`; apagado após export |
| Master guarda tudo | `ImportFindingsJSON` → `var/findings/` local |
| Scanned só após devolução | `MarkScannedForWorker` corre depois do import OK |

## Log típico

```
Fonte única: findings e domínios ficam apenas no goscan local
Partição: cada domínio pendente vai para exactamente 1 destino (de 2)
  · Local: batch de 2940000 domínios únicos
  · VPS-EU: batch de 2940000 domínios únicos
[VPS-EU] a enviar batch de 2940000 domínios únicos…
[VPS-EU] a testar batch (50 threads)…
[VPS-EU] a devolver findings ao orchestrador…
[VPS-EU] a fundir findings na base local…
```

## Configuração

1. **Settings → Workers remotos** — SSH ou HTTP API
2. **Settings → Deploy remoto** — URL do repo privado (recomendado para vários VPS)
3. **Scan → destinos** — Local + filhos (ou só filhos)
4. **Deploy/update** — instala binário no filho; findings nunca ficam lá

Ver [scan-hub.md](./scan-hub.md) — hub WebSocket com conteúdo `.env` encriptado em tempo real.

## CLI (filho manual)

```bash
# Worker HTTP efémero (orchestrador usa ephemeral:true)
goscan worker -listen :9090 -token TOKEN

# Export para o master fundir
goscan findings export-json -run-id RUN_ID -db /tmp/.../dominios.db -findings /tmp/.../findings
```
