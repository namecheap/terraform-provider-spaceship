data "spaceship_domain_list" "all" {}

output "total_domains" {
  value = data.spaceship_domain_list.all.total
}

output "first_domain" {
  value = length(data.spaceship_domain_list.all.items) > 0 ? {
    name             = data.spaceship_domain_list.all.items[0].name
    lifecycle_status = data.spaceship_domain_list.all.items[0].lifecycle_status
    nameservers      = data.spaceship_domain_list.all.items[0].nameservers.hosts
  } : null
}
