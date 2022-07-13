# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Create a new snapshot
resource "contabo_instance_snapshot" "snapshotInstance42" {
  name        = "snapshot-of-instance"
	description = "snapshot of the instance with id 42"
	instance_id = 42
}

# Update an existing snapshot
resource "contabo_instance_snapshot" "snapshotInstance42" {
	description = "terraform test-snapshot"
}

