SHELL := /bin/bash

.PHONY: test
test:
	go test -run 'Test[^A]' ./...

.PHONY: testacc
testacc:
	@if [[ -z "$$SPACESHIP_API_KEY" ]]; then \
		echo "SPACESHIP_API_KEY must be set"; exit 1; \
	fi
	@if [[ -z "$$SPACESHIP_API_SECRET" ]]; then \
		echo "SPACESHIP_API_SECRET must be set"; exit 1; \
	fi
	@if [[ -z "$$SPACESHIP_TEST_DOMAIN" ]]; then \
		echo "SPACESHIP_TEST_DOMAIN must be set"; exit 1; \
	fi
	go test -run TestAcc ./internal/provider -v

.PHONY: docs
docs:
	@go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	@PATH="$(shell go env GOPATH)/bin:$$PATH" tfplugindocs generate --provider-name "spaceship"

.PHONY: docs-validate
docs-validate:
	@go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	@PATH="$(shell go env GOPATH)/bin:$$PATH" tfplugindocs validate --provider-name "spaceship"