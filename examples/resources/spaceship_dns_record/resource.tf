# Manage a single DNS record. Use this when you want to manage records
# one-by-one. To own the entire custom record set for a domain in a single
# resource, use spaceship_dns_records instead.
#
# Caution: do not mix this resource with spaceship_dns_records for the same
# domain. The plural resource will delete any record not in its list,
# including records created by this singular resource. Pick one or the
# other per domain.
#
# The Spaceship API has no "update record data" operation: it matches records
# by (type, name, data). Changing any field other than `ttl` therefore forces
# the record to be destroyed and recreated. `lifecycle { create_before_destroy }`
# is recommended so the replacement record is added before the old one is
# removed, avoiding a window where the host is unresolvable.

resource "spaceship_dns_record" "web" {
  domain  = "example.com"
  type    = "A"
  name    = "@"
  ttl     = 3600
  address = "203.0.113.10"

  lifecycle {
    # Rotate the new record in before tearing the old one down. The DNS zone
    # briefly holds both, which is the desired behavior for an IP rotation.
    create_before_destroy = true
  }
}

# `ttl` is the one field that is updated in place — no replacement happens
# when only the TTL changes. Every other field (type, name, address, value,
# etc.) triggers Replace. Without `create_before_destroy`, replacement runs
# delete-then-create, leaving a short gap where the record does not exist.
