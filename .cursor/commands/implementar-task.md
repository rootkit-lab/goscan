---
related_skill: goscan-branch-task
---

# Implementar task

Implementar a **task activa** na **branch dedicada** (nunca em `main` directo).

## Fluxo

1. Ler `.tasks/README.md` — identificar task activa (prioridade P0/P1)
2. **Confirmar branch:** `git branch --show-current` = branch da task  
   Se estiveres em `main` → `git checkout <branch>` primeiro
3. Abrir `.tasks/<branch>.md` — checklist + **wireframes** (se UI)
4. Verificar código já feito (`git diff`, grep) — implementar **só** o que falta
5. **UI:** seguir wireframes da task; desvios → actualizar task ou pedir OK
6. Marcar checkboxes concluídos no `.md`
7. `/verify` (ou `make test` + `npm run build` se UI)
8. Actualizar `.tasks/README.md` se mudou prioridade/estado

## Regras

- Seguir convenções em `.cursor/rules/project-core.mdc`
- Checkers: `envutil.py` (`log_step`, `run_with_timeout`, `--batch`, `print_summary`)
- UI: tokens VS Code em `frontend/src/styles/`
- **Não commitar** sem pedido explícito
- Task completa → **`/integrar-main`** (não deixar branch pendente)

## Se não houver task activa

Sugerir `/criar-task` ou perguntar objectivo em uma frase.
