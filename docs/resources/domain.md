---
page_title: "spaceship_domain Resource - spaceship"
subcategory: ""
description: |-
  Manages domain settings for a Spaceship-managed domain.
---

# spaceship_domain (Resource)

Use this resource to manage domain settings for an existing Spaceship domain and store detailed metadata in state. Set `auto_renew` to enable or disable renewal automation, update nameservers when needed, and read lifecycle, verification, EPP, suspension, contact, and privacy details.

## Example Usage

```hcl
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

## Schema

### Required

- `domain` (String) Domain name to read from Spaceship.

### Optional

- `auto_renew` (Boolean) Whether the domain renews automatically. When omitted, the current setting is preserved.
- `nameservers` (Attributes) Nameserver settings for the domain. When omitted, the current nameserver configuration is preserved.

### Read-Only

- `name` (String) Domain name in ASCII format (A-label).
- `unicode_name` (String) Domain name in UTF-8 format (U-label).
- `is_premium` (Boolean) Whether the domain is premium.
- `registration_date` (String) Domain registration date as reported by Spaceship.
- `expiration_date` (String) Domain expiration date as reported by Spaceship.
- `lifecycle_status` (String) Lifecycle phase. One of creating, registered, grace1, grace2, redemption.
- `verification_status` (String) Status of the RAA verification process. One of verification, success, failed. Null when not applicable.
- `epp_statuses` (List of String) EPP status codes such as clientDeleteProhibited, clientHold, clientRenewProhibited, clientTransferProhibited, and clientUpdateProhibited.
- `suspensions` (List of Object) Suspension details (up to 2 items).
- `contacts` (Attributes) Contact handles and attributes returned by Spaceship.
- `privacy_protection` (Attributes) WHOIS privacy status for the domain.

### Nested Schema for `nameservers`

#### Optional

- `provider` (String) Nameserver provider type. Allowed values: `basic` or `custom`.
- `hosts` (Set of String) Nameserver hostnames. Required when `provider` is `custom` and must contain 2 to 12 entries. Must be omitted when `provider` is `basic`. The default Spaceship nameservers (`launch1.spaceship.net`, `launch2.spaceship.net`) can only be used with `provider = "basic"`.

### Nested Schema for `suspensions`

#### Read-Only

- `reason_code` (String) Suspension reason code (raaVerification, abuse, promoAbuse, fraud, pendingAccountVerification, unauthorizedAccess, tosViolation, transferDispute, restrictedSecurity, lockCourt, suspendCourt, udrpUrs, restrictedLegal, paymentPending, unpaidService, restrictedWhois, lockedWhois).

### Nested Schema for `contacts`

#### Read-Only

- `registrant` (String) Registrant contact handle.
- `admin` (String) Administrative contact handle when provided.
- `tech` (String) Technical contact handle when provided.
- `billing` (String) Billing contact handle when provided.
- `attributes` (List of String) Optional list of contact attributes supplied by Spaceship.

### Nested Schema for `privacy_protection`

#### Read-Only

- `contact_form` (Boolean) Indicates whether WHOIS should display the contact form link.
- `level` (String) Privacy level: `public` or `high`.

## Import

Import an existing domain by specifying the domain name:

```shell
terraform import spaceship_domain.example example.com
```
