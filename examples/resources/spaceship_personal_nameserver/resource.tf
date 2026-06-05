# `host` is the label relative to `domain` (e.g. "ns1"), not an FQDN.
# `ips` must be real, public, routable addresses — reserved ranges are rejected.

resource "spaceship_personal_nameserver" "ns1" {
  domain = "example.com"
  host   = "ns1"
  ips    = ["198.51.100.10", "198.51.100.11"]
}

resource "spaceship_personal_nameserver" "ns2" {
  domain = "example.com"
  host   = "ns2"
  ips    = ["198.51.100.20"]
}
