resource "spaceship_dns_record" "web" {
  domain  = "example.com"
  type    = "A"
  name    = "@"
  ttl     = 3600
  address = "203.0.113.10"

  lifecycle {
    # Add the replacement record before removing the old one; any change
    # other than `ttl` replaces the record.
    create_before_destroy = true
  }
}
