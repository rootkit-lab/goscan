---
related_skill: goscan-branch-task
---

# Implementar task

Implementar a **task activa** do backlog local.

## Fluxo

1. Ler `.tasks/README.md` — identificar task activa (prioridade P1)
2. Abrir `.tasks/<branch>.md` — checklist completo + wireframes
3. Verificar duplicados / código já feito (`git diff`, grep)
4. Implementar **só** o que falta na checklist (diff mínimo)
5. Marcar checkboxes concluídos no `.md` da task
6. Correr `/verify` (ou `make test` + `npm run build`)
7. Actualizar `.tasks/README.md` se a task ficar concluída

## Regras

- Seguir convenções em `.cursor/rules/project-core.mdc`
- Checkers: `envutil.py` (`log_step`, `run_with_timeout`, `--batch`, `print_summary`)
- UI: tokens VS Code em `frontend/src/styles/`
- **Não commitar** sem pedido explícito do utilizador
- Responder em português; resumo final curto (não listar cada ficheiro — Sidy agrupa)

## Se não houver task activa

Sugerir `/criar-task` ou perguntar objectivo em uma frase.
