resource "spaceship_domain" "example" {
  domain     = "example.com"
  auto_renew = true

  nameservers = {
    provider = "custom"
    hosts = [
      "ns1.example.net",
      "ns2.example.net",
    ]
  }
}
