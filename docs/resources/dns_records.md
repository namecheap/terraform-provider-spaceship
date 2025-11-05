---
page_title: "spaceship_dns_records Resource - spaceship"
subcategory: ""
description: |-
  Manages the complete DNS record set for a Spaceship-managed domain.
---

# spaceship_dns_records (Resource)

Manage the full DNS record set for an existing Spaceship domain. The Spaceship API applies DNS updates as a batch, so each Terraform apply replaces the entire record list with the configuration declared in this resource.

## Example Usage

```hcl
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
      type       = "MX"
      name       = "@"
      ttl        = 3600
      exchange   = "mail.example.com"
      preference = 10
    }
  ]
}
```

> **Note:** The Spaceship API expects the full authoritative set of records for the domain. Any records omitted from `records` are deleted during apply. The `force` argument defaults to `true` to satisfy Spaceship's overwrite requirement.

## Schema

### Required

- `domain` (String) Domain name to manage (for example `example.com`). The domain must already exist in the Spaceship account.
- `records` (Attributes List) Complete list of DNS records that should exist for the domain. Each apply replaces the entire set.

### Optional

- `force` (Boolean) Force Spaceship to apply the DNS update even if conflicts are detected. Defaults to `true`.

### Read-Only

- `id` (String) Domain identifier used in Terraform state (mirrors `domain`).

### Nested Schema for `records`

The `records` attribute is a list of nested objects representing individual DNS records.

#### Required

- `name` (String) Record host. Use `@` for the apex.
- `type` (String) DNS record type. Allowed values: `A`, `AAAA`, `ALIAS`, `CAA`, `CNAME`, `HTTPS`, `MX`, `NS`, `PTR`, `SRV`, `SVCB`, `TLSA`, `TXT`.

#### Optional

- `address` (String) IPv4 or IPv6 address for `A` and `AAAA` records.
- `alias_name` (String) Alias target for `ALIAS` records.
- `association_data` (String) Hex-encoded association data for `TLSA` records.
- `cname` (String) Canonical name for `CNAME` records.
- `exchange` (String) Mail exchange host for `MX` records.
- `flag` (Number) Flag for `CAA` records (`0` or `128`).
- `matching` (Number) Matching type for `TLSA` records (`0`-`255`).
- `nameserver` (String) Nameserver host for `NS` records.
- `pointer` (String) Pointer target for `PTR` records.
- `port` (String) Port for `HTTPS`, `SVCB`, and `TLSA` records. Must be `*` or an underscore followed by digits (for example `_443`).
- `port_number` (Number) Port for `SRV` records (`1`-`65535`).
- `preference` (Number) Preference for `MX` records (`0`-`65535`).
- `priority` (Number) Priority for `SRV` records (`0`-`65535`).
- `protocol` (String) Protocol label for `SRV`/`TLSA` records (for example `_tcp`).
- `scheme` (String) Scheme label for `HTTPS`, `SVCB`, and `TLSA` records (for example `_https`).
- `selector` (Number) Selector for `TLSA` records (`0`-`255`).
- `service` (String) Service label for `SRV` records (for example `_sip`).
- `svc_params` (String) SvcParams string for `HTTPS`/`SVCB` records.
- `svc_priority` (Number) Service priority for `HTTPS`/`SVCB` records (`0`-`65535`).
- `tag` (String) Tag for `CAA` records (for example `issue`).
- `target` (String) Target host for `SRV` records.
- `target_name` (String) Target name for `HTTPS`/`SVCB` records.
- `ttl` (Number) Record TTL in seconds (defaults to `3600`). Valid range: `60`-`3600`.
- `usage` (Number) Usage value for `TLSA` records (`0`-`255`).
- `value` (String) Value field used by several record types (`CAA`, `TXT`).
- `weight` (Number) Weight for `SRV` records (`0`-`65535`).

## Import

Import existing DNS records for a domain by specifying the domain name:

```shell
terraform import spaceship_dns_records.root example.com
```
