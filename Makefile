SHELL := /bin/bash

BINARY := terraform-provider-spaceship
GOBIN  := $(shell go env GOBIN)
GOBIN  := $(if $(GOBIN),$(GOBIN),$(shell go env GOPATH)/bin)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the provider binary
	go build -o $(BINARY) .

.PHONY: install
install: ## Build and install for local dev_overrides (prints the ~/.terraformrc block)
	go install .
	@echo ""
	@echo "Installed $(BINARY) to $(GOBIN)"
	@echo "Add this to ~/.terraformrc to use the local build (no 'terraform init' needed):"
	@echo ""
	@echo '  provider_installation {'
	@echo '    dev_overrides {'
	@echo '      "namecheap/spaceship" = "$(GOBIN)"'
	@echo '    }'
	@echo '    direct {}'
	@echo '  }'

.PHONY: test
test: ## Run unit tests (excludes acceptance)
	go test -skip 'TestAcc' ./...

.PHONY: test-cover
test-cover: ## Run unit tests with a coverage report
	go test -skip 'TestAcc' -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

.PHONY: testacc
testacc: ## Run acceptance tests against the real Spaceship API (loads .env if present)
	@set -a; [[ -f .env ]] && source .env; set +a; \
	for v in SPACESHIP_API_KEY SPACESHIP_API_SECRET SPACESHIP_TEST_DOMAIN; do \
		if [[ -z "$${!v}" ]]; then \
			echo "$$v must be set — export it or add it to .env (see .env.example)"; exit 1; \
		fi; \
	done; \
	TF_ACC=1 go test -run TestAcc ./internal/provider -v -count=1 -timeout=30m -failfast

.PHONY: lint
lint: ## Run linters and the formatter check
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Auto-fix formatting
	golangci-lint fmt ./...

.PHONY: docs
docs: ## Generate provider documentation
	@go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	@PATH="$(shell go env GOPATH)/bin:$$PATH" tfplugindocs generate --provider-name "spaceship"

.PHONY: docs-validate
docs-validate: ## Validate docs match the schema
	@go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	@PATH="$(shell go env GOPATH)/bin:$$PATH" tfplugindocs validate --provider-name "spaceship"
