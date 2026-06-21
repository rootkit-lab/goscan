# Verify

Verificação rápida antes de commit ou fim de sessão — **corre tudo**, corrige falhas encontradas.

## Checklist automático

1. `make test`
2. `make build`
3. `scripts/.venv/bin/python -m py_compile scripts/envutil.py scripts/chk-*.py`
4. `cd src/goscan-ui/frontend && npm run build`

## Se algo falhar

- Corrigir só o necessário (diff mínimo)
- Re-correr o passo que falhou
- Não commitar a menos que o utilizador peça

## Relatório final (curto)

- ✅ / ❌ por passo
- Ficheiros alterados (se houve fix)
- Comando sugerido a seguir (`make dev-ui` ou `/implementar-task`)
