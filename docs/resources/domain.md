---
page_title: "spaceship_domain Resource - spaceship"
subcategory: ""
description: |-
  Manage domain-level settings such as auto-renewal, privacy protection, and nameservers for a Spaceship-managed domain.
---

# spaceship_domain (Resource)

Manage high-level domain settings for an existing Spaceship domain, including auto-renewal, WHOIS privacy, and nameserver configuration. The resource reads full domain metadata (contacts, EPP statuses, suspensions) into state for visibility.

## Example Usage

Set privacy preferences and switch to custom nameservers:

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

Reset to Spaceship-managed nameservers:

```hcl
resource "spaceship_domain" "this" {
  domain = "example.com"

  nameservers = {
    provider = "basic" # hosts are ignored for basic
  }
}
```

> **Note:** When `provider` is `basic`, the Spaceship API ignores `hosts` and expects them to be omitted. The provider automatically drops `hosts` for the `basic` provider.

## Schema

### Required

- `domain` (String) Domain name to manage (for example `example.com`). Must already exist in the Spaceship account.

### Optional

- `auto_renew` (Boolean) Enable automatic renewal for the domain.
- `nameservers` (Attributes) Nameserver configuration.
  - `provider` (String) Nameserver provider. One of `basic` or `custom`.
  - `hosts` (List of String) Nameserver hostnames. Required when `provider` is `custom`; ignored when `provider` is `basic`. Must contain between 2 and 12 FQDNs.
- `privacy_protection` (Attributes) WHOIS privacy preferences.
  - `contact_form` (Boolean) Whether WHOIS should display a contact form link.
  - `level` (String) Privacy level. One of `public` or `high`.

### Read-Only

- `name` (String) Domain name (ASCII).
- `unicode_name` (String) Domain name in Unicode.
- `is_premium` (Boolean) Whether the domain is premium.
- `registration_date` (String) Registration timestamp.
- `expiration_date` (String) Expiration timestamp.
- `lifecycle_status` (String) Domain lifecycle status (creating, registered, grace1, grace2, redemption).
- `verification_status` (String) RAA verification status (verification, success, failed).
- `epp_statuses` (List of String) EPP status codes applied to the domain.
- `suspensions` (Attributes List) Domain suspensions.
  - `reason_code` (String) Suspension reason code.
- `contacts` (Attributes) Domain contacts.
  - `registrant` (String) Registrant contact handle.
  - `admin` (String) Administrative contact handle.
  - `tech` (String) Technical contact handle.
  - `billing` (String) Billing contact handle.
  - `attributes` (List of String) Optional contact attributes.
- `privacy_protection` (Attributes) Current privacy settings as reported by Spaceship.
- `nameservers` (Attributes) Current nameserver settings as reported by Spaceship.

## Import

Import an existing domain by its name:

```shell
terraform import spaceship_domain.this example.com
```
