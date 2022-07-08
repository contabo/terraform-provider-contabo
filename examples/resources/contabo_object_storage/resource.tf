# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user = "[your username]"
  oauth2_pass = "[your password]"
}

# Create a new object storage in region EU
resource "contabo_object_storage" "object_storage_eu" {
  region                   = "EU"
	total_purchased_space_tb = 2
}

# Update a new object storage, enable autoscaling
resource "contabo_object_storage" "object_storage_eu" {
  auto_scaling {
    state         = "enabled"
    size_limit_tb = 5
  }
}
