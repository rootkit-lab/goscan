# goscan-worker-releases

Repositório **privado** só com o binário `goscan-remote` para instalação rápida em VPS.

Estrutura publicada por `make publish-worker`:

```
linux-amd64/
  goscan-remote
  VERSION
  BINHASH
install-worker.sh
README.md
```

Tags git: `v{VERSION}` (ex.: `v1.0.0`).

## Instalação manual no VPS

```bash
git clone --depth 1 --branch v1.0.0 git@github.com:SEU_USER/goscan-worker-releases.git
cd goscan-worker-releases
./install-worker.sh
```

## Acesso privado

- **SSH (recomendado):** deploy key read-only no GitHub/GitLab; URL `git@github.com:USER/goscan-worker-releases.git`
- **HTTPS:** token read-only; configurar na UI GoScan (Settings → Deploy remoto)

O orchestrador GoScan faz `git clone`/`fetch` via SSH nos workers quando o repo está configurado.
