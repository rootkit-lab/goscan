# Skill: goscan-branch-task

## Quando usar

Nova feature, fix ou chore no goscan — sempre com **branch dedicada** + task local.

## Ciclo de vida

```
/criar-task  →  branch + .tasks/<branch>.md  (+ wireframes se UI)
      ↓
/implementar-task  →  código na branch, checkboxes
      ↓
/verify  →  testes
      ↓
/integrar-main  →  merge em main + arquivar task
```

## Fluxo `/criar-task`

1. Ler `.tasks/README.md` — evitar duplicados
2. Definir branch: `feat/…`, `fix/…` ou `chore/…`
3. `git checkout main && git pull` (se remoto) && `git checkout -b <branch>`
4. Copiar `TEMPLATE.md` → `.tasks/<branch>.md`
5. Preencher objectivo, escopo, checklist
6. **UI:** preencher secção **Wireframes** (ASCII obrigatório)
7. Actualizar `.tasks/README.md` (backlog)

## UI = wireframes obrigatórios

Considerar **UI** se a task menciona ou altera:

- `src/goscan-ui/`, `frontend/`, Wails, React, Tailwind
- sidebar, editor, painel, status bar, palette, botões, filtros visuais

Sem wireframes → task incompleta para implementação UI.

## Fluxo `/integrar-main`

1. Task na branch correcta; checkboxes críticos ✅
2. `/verify` verde (ou corrigir)
3. Commit(s) na branch se necessário
4. `git checkout main && git merge <branch>` (ou rebase — preferir merge local)
5. Arquivar: `.tasks/<branch>.md` → `.tasks/_archive/YYYY-MM/done/`
6. Actualizar `.tasks/README.md` (mover para Concluídas)
7. Opcional: `git branch -d <branch>`

## Arquivo

```
.tasks/
  README.md
  feat/minha-task.md      ← activa
  _archive/
    2026-06/
      done/
        feat/minha-task.md
```

## Paths

| Área | Path |
|------|------|
| CLI | `cmd/goscan/` |
| Scanner | `internal/scanner/` |
| Store | `internal/store/` |
| UI | `src/goscan-ui/` |
| Scripts | `scripts/` |
