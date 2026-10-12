GO ?= $(shell command -v go || echo /usr/local/go/bin/go)
GOROOT := $(shell $(GO) env GOROOT)
GOPATH := $(shell $(GO) env GOPATH)

# Node comes from nvm, at the major version in web/.nvmrc. nvm only loads in
# interactive shells, so make (and the git and Cursor hooks that call it) put
# the newest installed matching version first on PATH. Prepended, not appended,
# so it wins over any Windows npm that WSL adds to PATH.
NODE_MAJOR := $(shell cat web/.nvmrc 2>/dev/null)
NVM_DIR ?= $(HOME)/.nvm
NODE_BIN := $(shell ls -d $(NVM_DIR)/versions/node/v$(NODE_MAJOR).*/bin 2>/dev/null | sort -V | tail -n 1)

export PATH := $(if $(NODE_BIN),$(NODE_BIN):)$(PATH):$(GOROOT)/bin:$(GOPATH)/bin

SQLC := $(GO) tool sqlc
LEFTHOOK := $(GO) tool lefthook
GOOSE := $(GO) tool goose

GOOSE_DRIVER := postgres
GOOSE_MIGRATION_DIR := db/migrations
GOOSE_DBSTRING ?= postgres://blog:blog@localhost:5432/blog?sslmode=disable

# Seed file for `make seed`: a name in db/seed without the .sql suffix, e.g. SEED=posts-tags.
SEED_DIR := db/seed
SEED ?= posts-tags

# Files for fmt targets; override with FILES="a.go b.go".
# Note: FILES is expanded unquoted, so paths containing spaces are unsupported.
FILES ?= .

# Paths for `make import`, e.g. POSTS="content/a.md content/".
POSTS ?=

.PHONY: sqlc sqlc-check lint fmt fmt-check test hooks migrate migrate-down migrate-status migrate-new up down restart seed import api node-check

up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose down
	docker compose up -d

migrate:
	$(GOOSE) $(GOOSE_DRIVER) -dir $(GOOSE_MIGRATION_DIR) $(GOOSE_DBSTRING) up

migrate-down:
	$(GOOSE) $(GOOSE_DRIVER) -dir $(GOOSE_MIGRATION_DIR) $(GOOSE_DBSTRING) down

migrate-status:
	$(GOOSE) $(GOOSE_DRIVER) -dir $(GOOSE_MIGRATION_DIR) $(GOOSE_DBSTRING) status

migrate-new:
	$(GOOSE) $(GOOSE_DRIVER) -dir $(GOOSE_MIGRATION_DIR) $(GOOSE_DBSTRING) create $(NAME) sql

seed:
	@if [ -z "$(SEED)" ] || [ ! -f "$(SEED_DIR)/$(SEED).sql" ]; then \
		echo "usage: make seed SEED=<name>"; \
		echo "available seeds:"; \
		ls $(SEED_DIR) | sed -n 's/\.sql$$//p' | sed 's/^/  /'; \
		exit 1; \
	fi
	docker compose exec -T postgres psql "$(GOOSE_DBSTRING)" -v ON_ERROR_STOP=1 < $(SEED_DIR)/$(SEED).sql

api:
	DATABASE_URL="$(GOOSE_DBSTRING)" $(GO) run ./cmd/api

import:
	@if [ -z "$(POSTS)" ]; then echo 'usage: make import POSTS="content/a.md content/"'; exit 2; fi
	DATABASE_URL="$(GOOSE_DBSTRING)" $(GO) run ./cmd/blogctl import $(POSTS)

sqlc:
	@if [ -f sqlc.yaml ]; then $(SQLC) generate; else echo "sqlc.yaml not found, skipping sqlc generate"; fi

sqlc-check:
	@if [ -f sqlc.yaml ]; then $(SQLC) diff; else echo "sqlc.yaml not found, skipping sqlc diff"; fi

lint:
	golangci-lint run ./...

fmt:
	gofmt -w $(FILES)

fmt-check:
	@out=$$(gofmt -l $(FILES)); \
	if [ -n "$$out" ]; then \
		echo "These files need gofmt (run: make fmt FILES=<file>):"; \
		echo "$$out"; \
		exit 1; \
	fi

test:
	$(GO) test -race ./...

# Fail unless node and npm resolve to the nvm install of the major in web/.nvmrc.
node-check:
	@if [ -z "$(NODE_BIN)" ]; then \
		echo "no Node $(NODE_MAJOR).x under $(NVM_DIR); run: nvm install $(NODE_MAJOR)"; exit 1; \
	fi
	@if [ "$$(command -v node)" != "$(NODE_BIN)/node" ] || [ "$$(command -v npm)" != "$(NODE_BIN)/npm" ]; then \
		echo "node/npm resolve to $$(command -v node) and $$(command -v npm), want $(NODE_BIN)"; exit 1; \
	fi
	@echo "node $$(node --version), npm $$(npm --version) from $(NODE_BIN)"

# Install the git hooks defined in lefthook.yml (once per clone).
hooks:
	$(LEFTHOOK) install
