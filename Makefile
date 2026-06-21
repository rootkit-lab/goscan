SHELL := /bin/sh
ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
GO ?= go
NPM ?= npm
UI_DIR := src/goscan-ui
UI_FRONTEND := $(UI_DIR)/frontend
GOSCAN_UI_PORT ?= 9280
VERSION ?= $(shell tr -d '\n' < assets/VERSION 2>/dev/null || echo 0.0.0-dev)
LDFLAGS := -s -w

.PHONY: help build build-remote scan findings-list migrate-findings dev-ui setup-hooks test scripts-venv test-all-envs test-checkers-smoke batch-analyze release install uninstall install-doctor migrate-prod-data icon init-worker-release-repo publish-worker

help:
	@echo "goscan — targets:"
	@echo "  make build              # compila CLI"
	@echo "  make publish-worker     # publica goscan-remote num repo git privado"
	@echo "  make scan               # scan default (files/)"
	@echo "  make findings-list      # lista findings"
	@echo "  make migrate-findings   # migra scan_resultados_*"
	@echo "  make scripts-venv       # venv Python em scripts/.venv (checkers)"
	@echo "  make dev-ui             # Wails dev UI (dados no repo)"
	@echo "  make release            # build prod CLI + UI"
	@echo "  make install            # instala em ~/.local (prod)"
	@echo "  make migrate-prod-data  # copia repo → prod XDG"
	@echo "  make install-doctor     # verifica dev vs prod"
	@echo "  make test-all-envs      # testa checkers em batch (CLI; ARGS=--filter mysql)"
	@echo "  make batch-analyze      # analisa falhas do último batch"
	@echo "  make test-checkers-smoke # smoke: 1 execução por checker"
	@echo "  make setup-hooks        # instala githooks"

build: build-remote
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/goscan ./cmd/goscan

build-remote:
	$(GO) build -tags nosqlite -ldflags "$(LDFLAGS)" -o bin/goscan-remote ./cmd/goscan-remote

init-worker-release-repo:
	chmod +x scripts/init-worker-release-repo.sh
	./scripts/init-worker-release-repo.sh

publish-worker:
	chmod +x scripts/publish-worker-release.sh
	./scripts/publish-worker-release.sh

scan: build
	./bin/goscan -dir files

findings-list: build
	./bin/goscan findings list --limit 20

migrate-findings: build
	$(GO) run ./cmd/migrate-findings

SCRIPTS_VENV := scripts/.venv
SCRIPTS_PY := $(SCRIPTS_VENV)/bin/python
SCRIPTS_PIP := $(SCRIPTS_VENV)/bin/pip

scripts-venv:
	@if ! python3 -c "import venv" 2>/dev/null; then \
	  echo "Falta python3-venv. Instala: sudo apt install python3-venv python3-full"; \
	  exit 1; \
	fi
	@if [ ! -x "$(SCRIPTS_PY)" ]; then \
	  echo "A criar $(SCRIPTS_VENV)…"; \
	  python3 -m venv $(SCRIPTS_VENV); \
	  $(SCRIPTS_PIP) install -U pip; \
	fi
	@echo "A instalar dependências Python (scripts/requirements.txt)…"
	@$(SCRIPTS_PIP) install -r scripts/requirements.txt
	@echo "OK — checkers usam $(SCRIPTS_PY)"

dev-ui: scripts-venv
	@p=$(GOSCAN_UI_PORT); \
	for i in $$(seq 0 20); do \
	  cand=$$((p+i)); \
	  if ! ss -ltn 2>/dev/null | grep -q "127.0.0.1:$$cand "; then \
	    PORT=$$cand; break; \
	  fi; \
	done; \
	if [ -z "$$PORT" ]; then \
	  echo "Sem porta livre no intervalo $$p-$$((p+20)). Ex.: make dev-ui GOSCAN_UI_PORT=9400"; \
	  exit 1; \
	fi; \
	echo "dev-ui: usando porta $$PORT"; \
	cd $(UI_DIR) && GOSCAN_REPO_ROOT=$(ROOT_DIR) wails3 dev -config ./build/config.yml -port $$PORT

build-ui: build-ui-frontend
	$(GO) build -ldflags "-s -w" -o bin/goscan-ui ./src/goscan-ui

build-ui-frontend:
	cd $(UI_FRONTEND) && $(NPM) run build

icon:
	python3 scripts/icon-to-png.py

release: icon scripts-venv build-ui build
	@test -x bin/goscan-ui || (echo "❌ bin/goscan-ui em falta" && exit 1)
	@echo "Release $(VERSION) → bin/goscan bin/goscan-ui"

install: release
	chmod +x scripts/install.sh scripts/uninstall.sh scripts/install-doctor.sh
	./scripts/install.sh

uninstall:
	chmod +x scripts/uninstall.sh
	./scripts/uninstall.sh

install-doctor: build
	chmod +x scripts/install-doctor.sh
	./scripts/install-doctor.sh

migrate-prod-data:
	chmod +x scripts/migrate-dev-to-prod.sh
	./scripts/migrate-dev-to-prod.sh

setup-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/* 2>/dev/null || true

test:
	$(GO) test ./...

test-all-envs: build scripts-venv
	./bin/goscan test-all $(ARGS)

batch-analyze: build
	./bin/goscan batch-analyze $(ARGS)

test-checkers-smoke: scripts-venv
	$(SCRIPTS_PY) scripts/smoke-all-checkers.py
