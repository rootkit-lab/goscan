SHELL := /bin/sh
ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
GO ?= go
NPM ?= npm
UI_DIR := src/goscan-ui
UI_FRONTEND := $(UI_DIR)/frontend
GOSCAN_UI_PORT ?= 9280

.PHONY: help build scan findings-list migrate-findings dev-ui setup-hooks test scripts-venv

help:
	@echo "goscan — targets:"
	@echo "  make build              # compila CLI"
	@echo "  make scan               # scan default (files/)"
	@echo "  make findings-list      # lista findings"
	@echo "  make migrate-findings   # migra scan_resultados_*"
	@echo "  make scripts-venv       # venv Python em scripts/.venv (checkers)"
	@echo "  make dev-ui             # Wails dev UI"
	@echo "  make setup-hooks        # instala githooks"

build:
	$(GO) build -o bin/goscan ./cmd/goscan

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
	if ss -ltn 2>/dev/null | grep -q "127.0.0.1:$$p "; then \
	  echo "Porta $$p em uso. Feche o dev-ui anterior ou: make dev-ui GOSCAN_UI_PORT=9282"; \
	  exit 1; \
	fi
	cd $(UI_DIR) && GOSCAN_REPO_ROOT=$(ROOT_DIR) wails3 dev -config ./build/config.yml -port $(GOSCAN_UI_PORT)

build-ui: build-ui-frontend
	$(GO) build -o bin/goscan-ui ./src/goscan-ui

build-ui-frontend:
	cd $(UI_FRONTEND) && $(NPM) run build

setup-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/* 2>/dev/null || true

test:
	$(GO) test ./...
