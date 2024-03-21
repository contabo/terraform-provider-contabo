# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Search for a specific private network by ID
data "contabo_private_network" "testnetwork" {
  id = "1234"
}

output "my_test_private_network" {
  description = "my test private network"
  value = data.contabo_private_network.testnetwork
}