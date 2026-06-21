# Scan Hub — WebSocket seguro

Hub local que recebe **progresso** e **findings com conteúdo `.env` completo** dos workers remotos em tempo real.

## Fluxo

1. Orchestrator inicia `scanhub` em `127.0.0.1:PORT`
2. Por worker remoto: túnel SSH `-R` → `127.0.0.1:19280` no VPS → hub local
3. `goscan-remote` liga `ws://127.0.0.1:19280/hub` com token + AES-GCM
4. Cada `.env` HIGH → `SaveFinding` local imediato + evento UI
5. No fim: export JSON remoto reconcilia (idempotente)

## Segurança

- Token aleatório por worker/run
- Payloads encriptados **AES-256-GCM** (chave derivada do token via HKDF)
- Túnel SSH adicional; hub escuta só em localhost
- Conteúdo `.env` nunca aparece em logs UI

## Flags (`goscan-remote`)

```
-hub ws://127.0.0.1:19280/hub
-hub-token TOKEN
-worker-id worker-abc
```

## Fallback

Se o hub falhar, o scan continua com `@goscan/progress` no stderr (2 s) + export JSON no fim.

## Verificação

```bash
make build
make dev-ui
# Scan remoto → log: "Scan hub activo" · "hub conectado no filho"
# Findings aparecem na lista antes do fim do batch
```
