# Status do projecto

Snapshot rápido para orientar a próxima acção de dev.

## Recolher (correr em paralelo)

```bash
cd /home/wiz/Projects/goscan
git status -sb
git log -3 --oneline
ss -ltn 2>/dev/null | grep -E '9280|9282' || true
make findings-list 2>/dev/null | head -8
```

## Ler ficheiros

- `.tasks/README.md` — task activa e backlog
- `.tasks/feat/*.md` — checklist da task em curso (se existir)

## Resposta estruturada

1. **Branch / git** — alterações pendentes
2. **Task activa** — nome, % checklist estimado, próximo item
3. **Ambiente** — venv OK?, UI a correr?, porta
4. **Findings** — contagem breve (se CLI disponível)
5. **Sugestão** — um comando Cursor ou `make` para o próximo passo

Responde em português, conciso.
