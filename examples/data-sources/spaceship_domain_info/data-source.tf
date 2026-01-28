data "spaceship_domain_info" "example" {
  domain = "example.com"
}

output "domain_info" {
  value = {
    lifecycle_status    = data.spaceship_domain_info.example.lifecycle_status
    verification_status = data.spaceship_domain_info.example.verification_status
    expiration_date     = data.spaceship_domain_info.example.expiration_date
    nameservers         = data.spaceship_domain_info.example.nameservers.hosts
  }
}
