# Manage the entire custom DNS record set for a domain in one resource.
# On every apply this diffs the list against what's in the live zone and
# deletes any custom record not present here.
#
# Caution: because this resource owns the full custom group, do not mix it
# with spaceship_dns_record (singular) for the same domain. Records created
# by the singular resource will be deleted by this one. Pick one or the
# other per domain.

resource "spaceship_dns_records" "example" {
  domain = "example.com"

  records = [
    {
      type    = "A"
      name    = "@"
      ttl     = 3600
      address = "203.0.113.10"
    },
    {
      type    = "AAAA"
      name    = "@"
      ttl     = 3600
      address = "2001:db8::1"
    },
    {
      type       = "MX"
      name       = "@"
      ttl        = 3600
      exchange   = "mail.example.com"
      preference = 10
    },
    {
      type       = "ALIAS"
      name       = "@"
      ttl        = 3600
      alias_name = "origin.example.com"
    },
    {
      type  = "CNAME"
      name  = "www"
      ttl   = 3600
      cname = "example.com"
    },
    {
      type  = "TXT"
      name  = "@"
      ttl   = 3600
      value = "v=spf1 include:_spf.example.com ~all"
    }
  ]
}
