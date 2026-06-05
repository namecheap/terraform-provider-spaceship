> 🚧 **Active Development:** This provider is experimental, under active development, and not yet recommended for production workloads. Expect breaking changes and use only with disposable test domains.

# Spaceship Terraform Provider

[![CI](https://github.com/namecheap/terraform-provider-spaceship/actions/workflows/ci.yml/badge.svg)](https://github.com/namecheap/terraform-provider-spaceship/actions/workflows/ci.yml)
[![CodeQL](https://github.com/namecheap/terraform-provider-spaceship/actions/workflows/codeql.yml/badge.svg)](https://github.com/namecheap/terraform-provider-spaceship/actions/workflows/codeql.yml)
[![Release](https://img.shields.io/github/v/release/namecheap/terraform-provider-spaceship)](https://github.com/namecheap/terraform-provider-spaceship/releases)
[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/namecheap/spaceship)
[![Go Version](https://img.shields.io/github/go-mod/go-version/namecheap/terraform-provider-spaceship)](go.mod)
[![License](https://img.shields.io/github/license/namecheap/terraform-provider-spaceship)](LICENSE)
[![codecov](https://codecov.io/gh/namecheap/terraform-provider-spaceship/graph/badge.svg)](https://codecov.io/gh/namecheap/terraform-provider-spaceship)

This repository contains a Terraform provider that manages domain settings and DNS records for domains hosted with [Spaceship](https://spaceship.com/).

## Features

- Configure Spaceship credentials via provider configuration or environment variables.
- Read domain metadata and manage auto-renew and nameserver settings:
  - Name forms (ASCII and Unicode)
  - Premium flag
  - Registration and expiration dates
  - Lifecycle and verification status
  - EPP statuses
  - Suspensions
  - Contacts
  - Privacy protection
- Read the current DNS record set for an existing domain.
- Replace the full list of DNS records in a single Terraform apply.
- Enumerate every Spaceship-managed domain along with WHOIS, privacy, suspension, nameserver, and contact metadata via the `spaceship_domain_list` data source.

## Building

```bash
go build .
```

The resulting `terraform-provider-spaceship` binary can be copied to Terraform's plugin directory (`~/.terraform.d/plugins/<namespace>/spaceship/<version>/`) for local development. You can also run the provider directly using `terraform init` by setting the `TF_CLI_CONFIG_FILE` to point at a custom CLI configuration that defines a local dev override.

## Credentials

The provider authenticates against the Spaceship API using the same `X-API-Key` and `X-API-Secret` headers. Supply credentials either inline:

```hcl
provider "spaceship" {
  api_key    = "..."
  api_secret = "..."
}
```

or via the environment variables `SPACESHIP_API_KEY` and `SPACESHIP_API_SECRET`.

## Example Usage

```hcl
terraform {
  required_providers {
    spaceship = {
      source  = "registry.terraform.io/namecheap/spaceship"
      version = ">= 0.0.1"
    }
  }
}

provider "spaceship" {}

resource "spaceship_dns_records" "root" {
  domain = "example.com"

  records = [
    {
      type    = "A"
      name    = "@"
      ttl     = 3600
      address = "172.168.10.1"
    },
    {
      type  = "CNAME"
      name  = "www"
      ttl   = 3600
      cname = "example.com"
    },
    {
      type       = "MX"
      name       = "@"
      ttl        = 3600
      exchange   = "mail.example.com"
      preference = 10
    }
  ]
}

resource "spaceship_domain" "example" {
  domain     = "example.com"
  auto_renew = true

  nameservers = {
    provider = "custom"
    hosts = [
      "ns1.example.net",
      "ns2.example.net",
    ]
  }
}
```

## Domain Inventory Data Source

Use the `spaceship_domain_list` data source to inspect all domains in the Spaceship account and surface metadata in your configuration or outputs.

```hcl
data "spaceship_domain_list" "all" {}

output "first_domain" {
  value = {
    name        = data.spaceship_domain_list.all.items[0].name
    nameservers = data.spaceship_domain_list.all.items[0].nameservers.hosts
    privacy     = data.spaceship_domain_list.all.items[0].privacy_protection.level
  }
}
```

> **Note:** The Spaceship API updates DNS records in batches. Each apply **replaces** the full set of records for the domain with whatever is declared in Terraform. The resource's `force` argument defaults to `true` and may be adjusted if Spaceship changes its API requirements.

## Testing

Unit tests need no credentials and never touch the network:

```bash
make test
```

Acceptance tests exercise the full Terraform lifecycle against the **real** Spaceship API, so they need live credentials and a disposable test domain. Following the Terraform convention, they read configuration from environment variables and the framework only runs them when `TF_ACC` is set.

`make testacc` loads those variables from a gitignored `.env` file so you don't have to export them by hand. Copy the template and fill it in:

```bash
cp .env.example .env
# then edit .env with your SPACESHIP_API_KEY / SPACESHIP_API_SECRET / SPACESHIP_TEST_DOMAIN
make testacc
```

| Variable | Required | Purpose |
| --- | --- | --- |
| `SPACESHIP_API_KEY` | yes | Spaceship API key |
| `SPACESHIP_API_SECRET` | yes | Spaceship API secret |
| `SPACESHIP_TEST_DOMAIN` | yes | Disposable domain the tests mutate |
| `SPACESHIP_TEST_RECORD_PREFIX` | no | Override the record-name prefix the tests use |
| `SPACESHIP_TEST_RECORD_NAME` | no | Override the record name the tests use |

`make testacc` sources `.env` and sets `TF_ACC=1` for you. You can also skip `.env` and export the variables manually — `make testacc` falls back to whatever is already in the environment. To run a single test directly, set `TF_ACC=1` yourself:

```bash
TF_ACC=1 go test -run TestAccFunctionName ./internal/provider -v
```

> ⚠️ Ensure the domain you supply is safe to modify. The acceptance tests create, update, import, and then destroy real DNS records on it.
