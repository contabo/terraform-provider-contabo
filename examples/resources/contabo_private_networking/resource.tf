# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user = "[your username]"
  oauth2_pass = "[your password]"
}

# Create a new private network
resource "contabo_private_network" "databasePrivateNetwork" {
  name        = "terraform-test-private-network"
	description = "terraform test private network"
	region 		= "EU"
  instance_ids = [42, 1000]
}

# Update a new private network
resource "contabo_private_network" "databasePrivateNetwork" {
  instance_ids = [42, 9521, 7312]
}
