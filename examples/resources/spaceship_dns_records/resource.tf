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
