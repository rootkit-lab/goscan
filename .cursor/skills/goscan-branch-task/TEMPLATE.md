# Task: {branch}

**Branch:** `{branch}`  
**Prioridade:** P?  
**Tipo:** feat | chore | fix

## Objectivo

(Uma frase clara — o que muda para o utilizador.)

## Escopo

| Incluído | Fora de escopo |
|----------|----------------|
| … | … |

## Wireframes

<!-- OBRIGATÓRIO se a task tocar UI (src/goscan-ui/, frontend, Wails, sidebar, painéis).
     Apagar esta secção só em tasks 100% backend/CLI/scripts. -->

### Layout geral

```
┌─────────────────────────────────────────────────────────────┐
│ [sidebar]          │ [editor / conteúdo]    │ [actions]    │
│                    │                        │              │
├────────────────────┴────────────────────────┴──────────────┤
│ [output / terminal]                                          │
├──────────────────────────────────────────────────────────────┤
│ status bar · mode · contagens                                │
└─────────────────────────────────────────────────────────────┘
```

### Ecrã / componente principal

```
┌─ Título secção ────────────────────────┐
│ [input / filtro]                       │
│ ┌────────────────────────────────────┐ │
│ │ item activo                        │ │
│ │ item                               │ │
│ └────────────────────────────────────┘ │
│ [Acção primária]  [Secundária]         │
└────────────────────────────────────────┘
```

### Estados

| Estado | Comportamento |
|--------|---------------|
| vazio | … |
| loading | … |
| erro | … |

### Tokens / referência

- Cores: tokens em `frontend/src/styles/` (VS Code dark)
- Densidade: 11–13px labels, monospace só editor/terminal

## Checklist

### 0. Branch

- [ ] `/criar-task` criou branch `{branch}` a partir de `main`
- [ ] `.tasks/{branch}.md` actualizado

### 1. Implementação

- [ ] …

### 2. Verificação

- [ ] `make test`
- [ ] `make build` (+ `npm run build` se UI)
- [ ] Teste manual descrito abaixo

## Ficheiros prováveis

```
(caminhos relativos ao repo)
```

## Critérios de aceitação

- [ ] …

## Verificação manual

```bash
make build
# + comandos específicos da task
```
