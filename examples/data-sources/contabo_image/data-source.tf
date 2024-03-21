# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Search for a specific image by ID
data "contabo_image" "debian_11" {
  id = "66abf39a-ba8b-425e-a385-8eb347ceac10"
}

output "my_test_image" {
  description = "my test image"
  value = data.contabo_image.debian_11
}