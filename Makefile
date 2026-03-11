SHELL := /bin/bash

GO ?= go
TOOLS_BIN := $(CURDIR)/.tools/bin
STATICCHECK := $(TOOLS_BIN)/staticcheck
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint
GOVULNCHECK := $(TOOLS_BIN)/govulncheck
export GOBIN := $(TOOLS_BIN)
export PATH := $(TOOLS_BIN):$(PATH)

.PHONY: verify fmt vet lint test build vulncheck tools check-go

verify: check-go fmt vet lint test build vulncheck

check-go:
	@version="$$( $(GO) env GOVERSION | sed 's/^go//' )"; \
	if [[ "$$(printf '%s\n1.24\n' "$$version" | sort -V | head -n1)" != "1.24" ]]; then \
		echo "Go 1.24 or newer is required (found $$version)"; \
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
	@./hack/run-golangci-lint.sh "$(GOLANGCI_LINT)"

test:
	@$(GO) test ./...
	@$(GO) test -race ./...

build:
	@$(GO) build ./...

vulncheck: tools
	@$(GOVULNCHECK) ./...

tools:
	@mkdir -p "$(TOOLS_BIN)"
	@test -x "$(STATICCHECK)" || $(GO) install honnef.co/go/tools/cmd/staticcheck@latest
	@if [[ ! -x "$(GOLANGCI_LINT)" ]] || ! "$(GOLANGCI_LINT)" --version 2>/dev/null | grep -q "version 2\\."; then \
		$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi
	@test -x "$(GOVULNCHECK)" || $(GO) install golang.org/x/vuln/cmd/govulncheck@latest
