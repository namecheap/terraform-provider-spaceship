# Contributing to Spaceship Terraform Provider

Thank you for your interest in contributing! This document explains how to get
started and what to expect during the review process.

## Reporting Bugs

Before opening a new issue, please [search existing issues](https://github.com/namecheap/terraform-provider-spaceship/issues)
to avoid duplicates. When filing a bug report, use the
[Bug Report](https://github.com/namecheap/terraform-provider-spaceship/issues/new?template=bug_report.yml) template and include:

- Terraform and provider versions
- A minimal Terraform configuration that reproduces the problem
- Expected vs actual behavior
- Relevant error output or debug logs

## Suggesting Enhancements

Use the [Feature Request](https://github.com/namecheap/terraform-provider-spaceship/issues/new?template=feature_request.yml) template.
Describe the use case and the behavior you expect. If the enhancement relates to
a specific Spaceship API endpoint, link to the relevant
[API documentation](https://docs.spaceship.dev/).

## Setting Up the Development Environment

### Prerequisites

- [Go](https://go.dev/doc/install) (see `go.mod` for the minimum version)
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- [golangci-lint](https://golangci-lint.run/welcome/install-locally/)

### Building

```bash
go build .
```

### Running the Provider Locally

Create or edit `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "namecheap/spaceship" = "/path/to/terraform-provider-spaceship"
  }
  direct {}
}
```

Then run `terraform plan` or `terraform apply` in a directory with a
configuration that uses the `spaceship` provider.

## Running Tests

### Unit Tests

```bash
make test
```

### Linting

```bash
golangci-lint run ./...
```

### Acceptance Tests

Acceptance tests run against the real Spaceship API and modify real DNS records.
**Only run them against a disposable test domain.**

```bash
export SPACESHIP_API_KEY="..."
export SPACESHIP_API_SECRET="..."
export SPACESHIP_TEST_DOMAIN="your-test-domain.com"
make testacc
```

## Submitting a Pull Request

1. Fork the repository and create a branch from `master`.
2. Make your changes. Follow the existing code style and patterns.
3. Add or update unit tests for any new or changed behavior.
4. Run `make test` and `golangci-lint run ./...` before pushing.
5. Open a pull request using the PR template. Reference any related issues.
6. Ensure CI passes. Acceptance tests run automatically when secrets are
   available.

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat: add support for SVCB records
fix: handle numeric port values in SRV records
test: add acceptance test for DNS record ordering
docs: update README with domain resource example
chore: bump Go version to 1.25.8
```

### What to Expect

- A maintainer will review your PR, usually within a few business days.
- Small, focused PRs are easier to review and more likely to be merged quickly.
- If changes are requested, push follow-up commits to the same branch.

## Code Organization

```
internal/
  client/     # HTTP client for the Spaceship API
  provider/   # Terraform resource and data source implementations
docs/         # Generated provider documentation (do not edit manually)
examples/     # Example Terraform configurations used by doc generation
```

- **Adding a new resource:** see the guide in [CLAUDE.md](CLAUDE.md#adding-a-new-resource).
- **Generated docs:** run `make docs` after schema changes and commit the result.
