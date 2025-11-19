> 🚧 **Active Development:** This provider is experimental, under active development, and not yet recommended for production workloads. Expect breaking changes and use only with disposable test domains.

# Spaceship Terraform Provider

This repository contains a Terraform provider that manages DNS records for domains hosted with [Spaceship](https://spaceship.com/).

## Features

- Configure Spaceship credentials via provider configuration or environment variables.
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

```bash
go test ./...
```

Unit tests cover the reconciliation helpers. Acceptance tests require real credentials and a disposable test domain:

```bash
export SPACESHIP_API_KEY=...
export SPACESHIP_API_SECRET=...
export SPACESHIP_TEST_DOMAIN=example.com
# Optional: export SPACESHIP_TEST_RECORD_PREFIX=tfacc
# Optional: export SPACESHIP_TEST_RECORD_NAME=tf-acc
go test -run TestAcc ./internal/provider -v
```

> ⚠️ Ensure the domain you supply is safe to modify. The acceptance test creates, updates, imports, and then destroys a dedicated `A` record.
