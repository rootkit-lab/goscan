---
related_skill: goscan-branch-task
---

# Criar task

Cria uma task local **e a branch dedicada** — todo o trabalho da feature fica isolado até `/integrar-main`.

## Entrada

O utilizador descreve o objectivo (ex.: «melhorar filtros na sidebar», «install prod», «novo checker X»).

## Fluxo (agente)

1. **Ler** `.tasks/README.md` — verificar duplicados / tasks sobrepostas
2. **Nomear branch** — `feat/…`, `fix/…` ou `chore/…` (kebab-case, curto)
3. **Git** — trabalhar sempre na branch da task:
   ```bash
   git checkout main
   git checkout -b feat/nome-da-task
   ```
   (Se a branch já existir, fazer checkout em vez de criar.)
4. **Criar** `.tasks/<branch>.md` a partir de `.cursor/skills/goscan-branch-task/TEMPLATE.md`
5. **Preencher** objectivo, escopo, checklist, ficheiros prováveis, critérios de aceitação
6. **Wireframes (obrigatório se UI)** — ver regra abaixo
7. **Actualizar** `.tasks/README.md` — adicionar à tabela «Tasks activas» com prioridade
8. **Responder** em português: branch criada, ficheiro da task, próximo passo (`/implementar-task`)

## Regra: UI → wireframes

Se a task tocar **UI** (`src/goscan-ui/`, `frontend/`, Wails, sidebar, painéis, botões, filtros visuais, status bar):

- A secção **## Wireframes** é **obrigatória**
- Usar **ASCII** (layout geral + ecrã principal + tabela de estados)
- Referenciar tokens VS Code (`frontend/src/styles/`)
- **Não** implementar UI sem wireframes aprovados na task

Tasks só backend/CLI/scripts (`cmd/`, `internal/`, `scripts/`, Makefile) — apagar secção Wireframes do template.

## Conteúdo mínimo da task

| Secção | Obrigatório |
|--------|-------------|
| Objectivo | ✅ |
| Branch | ✅ |
| Escopo (in/out) | ✅ |
| Wireframes | ✅ se UI |
| Checklist §0 Branch | ✅ |
| Verificação | ✅ |

## Duplicados

- Mesmo tema que task activa → **não** criar nova; actualizar a existente
- Task concluída no código mas `.md` aberta → sugerir `/integrar-main`

## Não fazer

- Implementar código (isso é `/implementar-task`)
- Commitar sem pedido do utilizador
- Trabalhar em `main` directamente (excepto hotfix explícito)

## Exemplo

Pedido: «Painel de estatísticas na sidebar»

- Branch: `feat/sidebar-stats`
- Task: `.tasks/feat/sidebar-stats.md` com wireframe da sidebar + bloco stats
- README: P1 no backlog

## Fim do ciclo

Quando a task estiver pronta → **`/integrar-main`** (merge + arquivar).
