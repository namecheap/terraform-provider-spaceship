# CLAUDE.md

## Commands

```bash
go build .                                              # Build
make test                                               # Unit tests (excludes acceptance)
go test -run TestFunctionName ./internal/provider       # Single unit test
golangci-lint run ./...                                 # Lint
make docs                                               # Generate docs
make docs-validate                                      # Validate docs match schema

# Acceptance tests — only run when explicitly asked
# Requires: SPACESHIP_API_KEY, SPACESHIP_API_SECRET, SPACESHIP_TEST_DOMAIN
make testacc
go test -run TestAccFunctionName ./internal/provider -v # Single acceptance test
```

## Verification workflow

After making changes, follow this order:

1. **Unit tests** — `make test` (or `go test -run TestName ./internal/provider` for a specific test)
2. **Lint** — `golangci-lint run ./...`
3. **Build** — `go build .`
4. **Acceptance tests** — run only when the user explicitly asks. These hit real APIs and modify real DNS records.

## Architecture

This is a [Terraform Plugin Framework v1](https://github.com/hashicorp/terraform-plugin-framework) provider (protocol 6) for managing Spaceship domains and DNS.

**`internal/provider`** — Terraform resource/data-source implementations. Each file implements one resource or data source using the Plugin Framework interfaces (`resource.Resource`, `datasource.DataSource`). Shared schema types, model builders, and reconciliation helpers live in `domain_common.go`. Custom validators (e.g. nameserver format) are in separate `_validator.go` files.

**`internal/client`** — HTTP client for the Spaceship API (`https://spaceship.dev/api/v1`). Handles authentication headers, request building, and error parsing. Each API domain (domains, DNS) has its own file with typed request/response methods.

**Key design decisions:**

- **DNS records use full-replacement**: `spaceship_dns_records` replaces the entire record set on every apply. `domain_common.go` contains the reconciliation logic that computes the diff between desired and actual state.
- **Rate-limit fallback**: The domain info endpoint falls back to the domain list API on HTTP 429 — preserve this pattern if modifying client code.
- **Nested attributes**: Single nested objects use `types.Object`; repeating blocks use `types.List` with `NestedAttributeObject`. Conversion helpers like `flattenNameservers()` and `buildDomainModel()` live in `domain_common.go`.

**Adding a new resource:**

1. Create `internal/provider/<name>_resource.go` implementing `resource.Resource`, `resource.ResourceWithConfigure`, and optionally `resource.ResourceWithImportState`.
2. Add the corresponding API methods in `internal/client/`.
3. Register the resource in `provider.go` → `Resources()`.
4. Add `examples/resources/<name>/resource.tf` and run `make docs`.

**Credentials:**

Provider reads `SPACESHIP_API_KEY` / `SPACESHIP_API_SECRET` env vars or inline HCL attributes. Auth is passed as `X-API-Key` and `X-API-Secret` HTTP headers.
