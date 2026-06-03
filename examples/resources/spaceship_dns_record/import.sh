# The composite ID has the form domain/TYPE/name/<data-signature>, with the
# four segments joined by "/". The <data-signature> is the record's
# type-specific data fields, lowercased and joined by "|" — for an A record
# that is a single field (just the IPv4 address, so no "|" appears), but other
# types have several, e.g. CAA is "flag|tag|value" and SRV is
# "service|protocol|priority|weight". See the resource's `id` documentation
# for the per-type field list.
#
# After a successful Create, `terraform state show <addr>` prints the exact
# ID — the easiest way to recover it for an import.
#
# You often don't need import at all: if a record with identical (type, name,
# data) already exists, just declare the resource and `terraform apply`. Create
# is idempotent and adopts the existing record, aligning its TTL to your
# config. Use import only to bring an existing record under management without
# issuing any write.

terraform import spaceship_dns_record.web "example.com/A/@/203.0.113.10"
