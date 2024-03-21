# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Create a new tag
resource "contabo_tag" "default_tag" {
  color = "#000002"
  name="NewTag"
}

# Update an existing tag
resource "contabo_tag" "default_tag" {
	color 		  = "#ffffff"
  name = "UpdatedTag"
}