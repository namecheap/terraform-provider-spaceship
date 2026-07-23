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

  # Optional: bound how long operations wait out API throttling.
  timeouts {
    create = "30m"
    read   = "10m"
    update = "30m"
  }
}
