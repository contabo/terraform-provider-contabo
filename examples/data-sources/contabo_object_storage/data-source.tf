# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Search object storage by ID
data "contabo_object_storage" "example1" {
  id = "3a6e5301-fc71-42ce-b60c-49841681c2da"
}

# Search object storage by display name
data "contabo_object_storage" "example2" {
  display_name = "example2"
}

output "my_object_storage1" {
  description = "my object storage 1"
  value = data.contabo_object_storage.example1
}

output "my_object_storage2" {
  description = "my object storage 2"
  value = data.contabo_object_storage.example1
}
