GO ?= $(shell command -v go || echo /usr/local/go/bin/go)
GOROOT := $(shell $(GO) env GOROOT)
GOPATH := $(shell $(GO) env GOPATH)
export PATH := $(PATH):$(GOROOT)/bin:$(GOPATH)/bin

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

.PHONY: sqlc sqlc-check lint fmt fmt-check test hooks migrate migrate-down migrate-status migrate-new up down restart seed

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

# Install the git hooks defined in lefthook.yml (once per clone).
hooks:
	$(LEFTHOOK) install
