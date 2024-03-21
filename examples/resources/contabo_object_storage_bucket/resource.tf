# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Create a new object storage
resource "contabo_object_storage" "example_object_storage" {
  region                   = "EU"
	total_purchased_space_tb = 0.500
}

# create a bucket in the object_storage
resource "contabo_object_storage_bucket" "example_bucket" {
  name                = "example_bucket"
  object_storage_id   = contabo_object_storage.example_object_storage.id
}