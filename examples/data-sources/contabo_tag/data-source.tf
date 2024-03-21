# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Get a specific tag by ID
data "contabo_tag" "default_tag" {
  id="26878"
}
output "output" {
  description = "output"
  value = data.contabo_tag.default_tag
}