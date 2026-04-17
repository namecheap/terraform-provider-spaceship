# DNS Records — Design Notes

## DNS Groups

The Spaceship API organizes DNS records into groups. Each record returned by the API may include a `group` object with a `type` field. There are exactly three group types:

| Group type   | Owner              | Managed by provider? |
|--------------|--------------------|----------------------|
| `custom`     | External API users | Yes                  |
| `product`    | Spaceship features (e.g. URL redirect) | No |
| `personalNS` | Personal nameservers | No                 |

The provider filters records in `GetDNSRecords()` via `filterCustomDNSRecords()` so that only `custom` (and ungrouped) records are visible to Terraform. Records in `product` and `personalNS` groups are never read, diffed, or deleted by the provider.

This means:

- `terraform plan` will not show drift for Spaceship-managed records.
- `terraform destroy` will only delete custom records, leaving feature-owned records intact.
- Users do not need to mirror Spaceship-managed records in their `.tf` files.

## Record Matching (Upsert API)

The `PUT /dns/records/{domain}` endpoint matches existing records by **type + name + data** using case-insensitive comparison for all fields, with one exception:

- **TXT records**: the `value` field is compared **case-sensitively**.

The provider's `recordValueSignature()` function follows the same rules. All fields are lowercased in the signature except TXT `value`, ensuring the provider's diff logic agrees with the API about what constitutes "the same record".

### Record type notes

- **ALIAS**: Resolves a canonical (true) domain name. Implements CNAME-like behavior for the zone apex where CNAME is not allowed. The `aliasName` field is a hostname (1-253 chars, `hostNameValue` pattern).

### Data fields per record type

| Type  | Data fields used for matching                          |
|-------|-------------------------------------------------------|
| A     | `address`                                              |
| AAAA  | `address`                                              |
| ALIAS | `aliasName`                                            |
| CAA   | `flag`, `tag`, `value`                                 |
| CNAME | `cname`                                                |
| HTTPS | `svcPriority`, `targetName`, `svcParams`, `port`, `scheme` |
| MX    | `exchange`, `preference`                               |
| NS    | `nameserver`                                           |
| PTR   | `pointer`                                              |
| SRV   | `service`, `protocol`, `priority`, `weight`            |
| SVCB  | `svcPriority`, `targetName`, `svcParams`, `port`, `scheme` |
| TLSA  | `port`, `protocol`, `usage`, `selector`, `matching`, `associationData` |
| TXT   | `value` (case-sensitive)                               |

When a match is found, only the TTL is updated. When no match is found, a new record is created.

## Reconciliation Flow

The provider uses **diff-based reconciliation**, not full-zone replacement. Only the minimal set of changes is sent to the API.

On `Create` and `Update`, the provider:

1. Fetches current custom records from the API (`GetDNSRecords` → filtered).
2. Computes a diff (`diffDNSRecords`):
   - Records in API but not in config → **delete** via `DELETE /dns/records/{domain}`.
   - Records in config but not in API (or with changed TTL) → **upsert** via `PUT /dns/records/{domain}`.
   - Records that match and have the same TTL → **no action** (left untouched).
3. Re-fetches records and reorders them to match the config ordering (for stable state).

The upsert API itself is also incremental: it matches incoming records against existing ones by type + name + data. If a match is found, only the TTL is updated. If no match is found, a new record is created. Unmentioned records are not deleted by the upsert call — that's why the provider sends a separate `DELETE` for removed records.

On `Delete`, the provider calls `ClearDNSRecords` which fetches all custom records and deletes them in a single request.
