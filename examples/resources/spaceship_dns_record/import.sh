# The composite ID has the form: domain/TYPE/name/<data-signature>
# The data signature is a normalized, pipe-separated representation of the
# record's type-specific fields. For an A record it is the lowercased IPv4
# address; for other types see the resource's `id` documentation.
#
# After a successful Create, `terraform state show <addr>` prints the exact
# ID — the easiest way to recover it for an import.

terraform import spaceship_dns_record.web "example.com/A/@/203.0.113.10"
