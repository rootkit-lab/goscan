# Skill: goscan-branch-task

## Quando usar

Nova branch / feature no goscan.

## Fluxo

1. Ler `.tasks/README.md`
2. Criar branch `feat/...` ou `chore/...`
3. Copiar `TEMPLATE.md` → `.tasks/<branch>.md`
4. Implementar com checkboxes
5. Pós-merge: arquivar em `.tasks/_archive/YYYY-MM/done/`

## Paths

| Área | Path |
|------|------|
| CLI | `cmd/goscan/` |
| Scanner | `internal/scanner/` |
| Store | `internal/store/` |
| UI | `src/goscan-ui/` |
| Scripts | `scripts/` |
