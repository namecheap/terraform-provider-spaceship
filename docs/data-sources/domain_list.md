---
page_title: "spaceship_domain_list Data Source - spaceship"
subcategory: ""
description: |-
  Lists every Spaceship domain in the account, including WHOIS, suspension, privacy, nameserver, and contact metadata.
---

# spaceship_domain_list (Data Source)

Use this data source to retrieve every domain that exists in the authenticated Spaceship account. The response includes WHOIS verification state, EPP statuses, suspension details, privacy protection settings, nameserver configuration, and contact handles for each domain.

## Example Usage

```hcl
data "spaceship_domain_list" "all" {}

output "domains" {
  value = {
    total  = data.spaceship_domain_list.all.total
    first  = data.spaceship_domain_list.all.items[0].name
    ns_set = data.spaceship_domain_list.all.items[0].nameservers.hosts
  }
}
```

## Schema

### Read-Only

- `items` (Attributes List) Domain entries returned by the Spaceship API, ordered by name. Each element is documented below.
- `total` (Number) Total number of domains in the Spaceship account.

### Nested Schema for `items`

#### Read-Only

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

- `hosts` (List of String) Fully-qualified nameserver hosts assigned to the domain.
- `provider` (String) Nameserver provider type. One of `basic` or `custom`.

### Nested Schema for `contacts`

#### Read-Only

- `admin` (String) Administrative contact handle, when available.
- `attributes` (List of String) Optional contact attributes supplied by Spaceship.
- `billing` (String) Billing contact handle, when available.
- `registrant` (String) Registrant contact handle (always present).
- `tech` (String) Technical contact handle, when available.
