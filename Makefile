SHELL := /bin/bash

GO ?= go
TOOLS_BIN := $(CURDIR)/.tools/bin
STATICCHECK := $(TOOLS_BIN)/staticcheck
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint
GOVULNCHECK := $(TOOLS_BIN)/govulncheck
export GOBIN := $(TOOLS_BIN)
export PATH := $(TOOLS_BIN):$(PATH)

STATICCHECK_VERSION ?= 2025.1.1
GOLANGCI_LINT_VERSION ?= v2.11.3
GOVULNCHECK_VERSION ?= v1.1.4

.PHONY: help verify fmt vet lint test build run vulncheck tools check-go

RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

ifneq ($(filter run,$(MAKECMDGOALS)),)
$(eval $(RUN_ARGS):;@:)
endif

help:
	@printf "Available targets:\n"
	@printf "  help       Show this help message\n"
	@printf "  verify     Run the full local verification suite\n"
	@printf "  fmt        Check Go formatting with gofmt\n"
	@printf "  vet        Run go vet\n"
	@printf "  lint       Run staticcheck and golangci-lint\n"
	@printf "  test       Run unit tests and race tests\n"
	@printf "  build      Build all packages\n"
	@printf "  run        Run the atb CLI with go run (use 'make run update' or ARGS='...')\n"
	@printf "  vulncheck  Run govulncheck\n"
	@printf "  tools      Install local toolchain dependencies into .tools/bin\n"

verify: check-go fmt vet lint test build vulncheck

check-go:
	@version="$$( $(GO) env GOVERSION | sed 's/^go//' )"; \
	if [[ "$$(printf '%s\n1.26\n' "$$version" | sort -V | head -n1)" != "1.26" ]]; then \
		echo "Go 1.26 or newer is required (found $$version)"; \
		exit 1; \
	fi

fmt:
	@out="$$(gofmt -l .)"; \
	if [[ -n "$$out" ]]; then \
		echo "$$out"; \
		exit 1; \
	fi

vet:
	@$(GO) vet ./...

lint: tools
	@$(STATICCHECK) ./...
	@$(GOLANGCI_LINT) run

test:
	@$(GO) test ./...
	@$(GO) test -race ./...

build:
	@$(GO) build ./...

run:
	@$(GO) run ./cmd/atb $(if $(ARGS),$(ARGS),$(RUN_ARGS))

vulncheck: tools
	@$(GOVULNCHECK) ./...

tools:
	@mkdir -p "$(TOOLS_BIN)"
	@test -x "$(STATICCHECK)" || $(GO) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	@if [[ ! -x "$(GOLANGCI_LINT)" ]] || ! "$(GOLANGCI_LINT)" --version 2>/dev/null | grep -q "$(GOLANGCI_LINT_VERSION)"; then \
		$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi
	@test -x "$(GOVULNCHECK)" || $(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
