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
