# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Search object storage bucket by Object Storage ID and name
data "contabo_object_storage_bucket" "bucket1" {
  object_storage_id = "3a6e5301-fc71-42ce-b60c-49841681c2da"
  name = "testbucket"
}

output "my_test_bucket" {
  description = "my test bucket"
  value = data.contabo_object_storage_bucket.bucket1
}