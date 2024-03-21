# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Search for a specific instance by ID
data "contabo_instance" "test_instance" {
  id = "123455"
}

output "my_test_instance" {
  description = "my test instance"
  value = data.contabo_instance.test_instance
}