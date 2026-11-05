# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Get a DNS zone by name
data "contabo_dns_zone" "example" {
  zone_name = "example.com"
}

output "name_servers" {
  value = data.contabo_dns_zone.example.name_servers
}
