---
page_title: "spaceship_domain Resource - spaceship"
subcategory: ""
description: |-
  Manages domain settings for a Spaceship-managed domain.
---

# spaceship_domain (Resource)

Use this resource to manage domain settings for an existing Spaceship domain and store the ASCII and Unicode forms of the name in state. Set `auto_renew` to enable or disable renewal automation.

## Example Usage

```hcl
resource "spaceship_domain" "example" {
  domain     = "example.com"
  auto_renew = true
}
```

## Schema

### Required

- `domain` (String) Domain name to read from Spaceship.

### Optional

- `auto_renew` (Boolean) Whether the domain renews automatically. When omitted, the current setting is preserved.

### Read-Only

- `name` (String) Domain name in ASCII format (A-label).
- `unicode_name` (String) Domain name in UTF-8 format (U-label).

## Import

Import an existing domain by specifying the domain name:

```shell
terraform import spaceship_domain.example example.com
```
