---
related_skill: goscan-branch-task
---

# Integrar main

Integra o trabalho da **branch da task** em `main`, arquiva a task local e actualiza o backlog.

## Pré-requisitos

- Task activa em `.tasks/<branch>.md`
- Estás na branch correcta **ou** sabes qual é (`git branch --show-current`)
- Código implementado; checkboxes críticos marcados

## Fluxo (agente)

1. **Identificar branch** — nome = ficheiro em `.tasks/feat/*.md` activo ou `git branch --show-current`
2. **Verificar** `.tasks/<branch>.md` — resumir o que foi feito vs checklist
3. **`/verify`** (ou equivalente):
   ```bash
   make test
   make build
   cd src/goscan-ui/frontend && npm run build   # se UI alterada
   ```
4. **Commit pendente?** — se `git status` dirty, pedir ao utilizador ou commitar **só se** pediu explicitamente antes
5. **Integrar em main:**
   ```bash
   git checkout main
   git merge <branch> -m "Merge branch '<branch>'"
   ```
   Se conflitos → resolver, `make test`, continuar merge.
6. **Arquivar task:**
   ```bash
   mkdir -p .tasks/_archive/$(date +%Y-%m)/done
   mv .tasks/<branch>.md .tasks/_archive/$(date +%Y-%m)/done/
   ```
7. **Actualizar** `.tasks/README.md`:
   - Remover de «Tasks activas»
   - Adicionar linha em «Concluídas»
   - Actualizar «Branch activa sugerida» para a próxima P0/P1
8. **Limpar branch** (opcional, se merge OK):
   ```bash
   git branch -d <branch>
   ```
9. **Prod?** — se a task incluiu install/release, sugerir `make release && make install`
10. **Responder** em português: merge OK, task arquivada, branch removida (se aplicável)

## Critérios para integrar

| OK | Bloquear |
|----|----------|
| `/verify` verde | Testes a falhar |
| Checklist core ✅ | Task UI sem wireframes na `.md` |
| Sem secrets no diff | `dominios.db`, `.env`, `var/` no commit |

## Se a task não estiver completa

- **Não** integrar — listar checkboxes em falta
- Sugerir `/implementar-task` ou `/continuar`

## Estrutura de arquivo

```
.tasks/_archive/2026-06/done/feat/minha-task.md
```

Tasks arquivadas são referência local (`.tasks/` não versionado).

## Relacionado

| Comando | Quando |
|---------|--------|
| `/criar-task` | Início — branch + task |
| `/implementar-task` | Durante — código |
| `/verify` | Antes de integrar |
| `/criar-release` | Depois — se mudou binários prod |
