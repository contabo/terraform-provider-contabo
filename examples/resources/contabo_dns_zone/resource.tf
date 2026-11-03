# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Host the DNS zone of example.com on the Contabo name servers
resource "contabo_dns_zone" "example" {
  zone_name = "example.com"
}

# Delegate the domain to these name servers at your registrar
output "name_servers" {
  value = contabo_dns_zone.example.name_servers
}
