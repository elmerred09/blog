GO ?= $(shell command -v go || echo /usr/local/go/bin/go)
GOROOT := $(shell $(GO) env GOROOT)
GOPATH := $(shell $(GO) env GOPATH)
export PATH := $(PATH):$(GOROOT)/bin:$(GOPATH)/bin

SQLC := $(GO) tool sqlc
LEFTHOOK := $(GO) tool lefthook

.PHONY: sqlc sqlc-check lint fmt fmt-check test hooks

# Files for fmt targets; override with FILES="a.go b.go".
# Note: FILES is expanded unquoted, so paths containing spaces are unsupported.
FILES ?= .

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
