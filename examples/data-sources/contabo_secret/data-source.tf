# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Search for a specific secret by ID
data "contabo_secret" "mysecret" {
  id = "123"
}

output "my_secret_output" {
  description = "my secret"
  value = data.contabo_secret.mysecret
}