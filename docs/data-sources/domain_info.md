---
page_title: "spaceship_domain_info Data Source - spaceship"
subcategory: ""
description: |-
  Fetches full details for a single Spaceship domain, including WHOIS, suspension, privacy, nameserver, and contact metadata.
---

# spaceship_domain_info (Data Source)

Use this data source to retrieve all metadata for one Spaceship domain. 
The response includes WHOIS verification state, EPP statuses, suspension details, privacy protection settings, nameserver configuration, and contact handles.

## Example Usage

```hcl
data "spaceship_domain_info" "example" {
  domain = "example.com"
}

output "domain_info" {
  value = {
    lifecycle = data.spaceship_domain_info.example.lifecycle_status
    ns_hosts  = data.spaceship_domain_info.example.nameservers.hosts
    contacts  = data.spaceship_domain_info.example.contacts
  }
}
```

## Schema

### Required

- `domain` (String) Domain name to look up.

### Read-Only

- `auto_renew` (Boolean) Whether the domain renews automatically.
- `contacts` (Attributes) Contact handles for each role. See [Contacts](#nested-schema-for-contacts).
- `epp_statuses` (List of String) List of EPP status codes assigned to the domain (for example `clientHold`, `clientTransferProhibited`).
- `expiration_date` (String) Expiration timestamp in ISO 8601 format.
- `is_premium` (Boolean) Indicates whether the domain is a premium registration.
- `lifecycle_status` (String) Lifecycle phase. Enum: `creating`, `registered`, `grace1`, `grace2`, `redemption`.
- `nameservers` (Attributes) Nameserver configuration for the domain. See [Nameservers](#nested-schema-for-nameservers).
- `name` (String) ASCII domain name.
- `privacy_protection` (Attributes) Privacy protection settings. See [Privacy Protection](#nested-schema-for-privacy_protection).
- `registration_date` (String) Registration timestamp in ISO 8601 format.
- `suspensions` (Attributes List) Suspension reasons returned by Spaceship, if any. See [Suspensions](#nested-schema-for-suspensions).
- `unicode_name` (String) Unicode/punycode representation of the domain.
- `verification_status` (String) RAA verification status. Enum: `verification`, `success`, `failed`. Null if the RAA procedure is not applied to the domain.

### Nested Schema for `suspensions`

#### Read-Only

- `reason_code` (String) One of the Spaceship suspension reason codes (for example `raaVerification`, `fraud`, `abuse`).

### Nested Schema for `privacy_protection`

#### Read-Only

- `contact_form` (Boolean) Whether WHOIS shows the Spaceship contact form.
- `level` (String) Privacy level reported by Spaceship. One of `public` or `high`.

### Nested Schema for `nameservers`

#### Read-Only

- `hosts` (Set of String) Fully-qualified nameserver hosts assigned to the domain.
- `provider` (String) Nameserver provider type. One of `basic` or `custom`.

### Nested Schema for `contacts`

#### Read-Only

- `admin` (String) Administrative contact handle, when available.
- `attributes` (List of String) Optional contact attributes supplied by Spaceship.
- `billing` (String) Billing contact handle, when available.
- `registrant` (String) Registrant contact handle (always present).
- `tech` (String) Technical contact handle, when available.
