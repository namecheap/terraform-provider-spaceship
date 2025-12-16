---
page_title: "Spaceship Provider"
subcategory: ""
description: |-
  The Spaceship provider enables Terraform configuration management for Spaceship domains and DNS records.
---

# Spaceship Provider

> 🚧 **Active Development:** This provider is still evolving and is not intended for production deployments. Expect breaking changes between releases and test in non-critical environments first.

Use the Spaceship provider to manage domains registered with Spaceship. Configure nameservers and WHOIS privacy, and replace the full DNS record set for a domain in a single Terraform operation using the Spaceship API (`https://spaceship.dev/api/v1`).

## Example Usage

```hcl
terraform {
  required_providers {
    spaceship = {
      source  = "namecheap/spaceship"
      version = ">= 0.0.1"
    }
  }
}

provider "spaceship" {
  api_key    = var.spaceship_api_key
  api_secret = var.spaceship_api_secret
}

resource "spaceship_dns_records" "root" {
  domain = "example.com"

  records = [
    {
      type    = "A"
      name    = "@"
      ttl     = 3600
      address = "203.0.113.10"
    },
    {
      type    = "MX"
      name    = "@"
      ttl     = 3600
      exchange   = "mail.example.com"
      preference = 10
    }
  ]
}
```

### Manage domain settings

```hcl
resource "spaceship_domain" "this" {
  domain = "example.com"

  auto_renew = false

  privacy_protection = {
    contact_form = false
    level        = "public"
  }

  nameservers = {
    provider = "custom"
    hosts = [
      "ns1.exampledomain.com",
      "ns2.exampledomain.com",
    ]
  }
}
```

## Authentication

Spaceship authenticates requests with an API key and secret. Configure credentials directly within the provider block or with environment variables:

- `SPACESHIP_API_KEY`
- `SPACESHIP_API_SECRET`

If both the configuration attribute and environment variable are set, the explicit configuration value takes precedence.

## Provider Configuration Reference

The following arguments are supported in the provider block. All attributes are optional if the equivalent environment variable is set.

- **api_key** (String) Spaceship API key. If omitted, the provider reads `SPACESHIP_API_KEY`. This value must be a non-empty string.
- **api_secret** (String) Spaceship API secret. If omitted, the provider reads `SPACESHIP_API_SECRET`. This value must be a non-empty string.

## Resources

The Spaceship provider currently offers the following resources:

- `spaceship_domain` — Manage domain-level settings including auto-renewal, privacy protection, and nameservers. The resource surfaces domain metadata such as contacts, suspensions, and EPP statuses.
- `spaceship_dns_records` — Manage the full DNS record set for a Spaceship-managed domain, replacing all records in each apply. The resource supports importing an existing domain by its name.

## Data Sources

- `spaceship_domain_info` — Retrieve WHOIS, privacy, suspension, nameserver, and contact metadata for a single Spaceship-managed domain.
- `spaceship_domain_list` — Retrieve every Spaceship-managed domain along with WHOIS, privacy, suspension, nameserver, and contact metadata.

## Import

Existing DNS configurations can be brought under Terraform management by importing the domain name into the provider's DNS records resource:

```shell
terraform import spaceship_dns_records.root example.com
```

The provider will read the current DNS records from Spaceship and populate Terraform state so that future plans accurately reflect drift.
