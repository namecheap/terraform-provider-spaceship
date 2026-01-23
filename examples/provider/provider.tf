terraform {
  required_providers {
    spaceship = {
      source  = "namecheap/spaceship"
      version = ">= 0.0.1"
    }
  }
}

provider "spaceship" {
  api_key    = var.spaceship_api_key
  api_secret = var.spaceship_api_secret
}

resource "spaceship_dns_records" "root" {
  domain = "example.com"

  records = [
    {
      type    = "A"
      name    = "@"
      ttl     = 3600
      address = "203.0.113.10"
    },
    {
      type       = "MX"
      name       = "@"
      ttl        = 3600
      exchange   = "mail.example.com"
      preference = 10
    }
  ]
}
