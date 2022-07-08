# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user = "[your username]"
  oauth2_pass = "[your password]"
}

# Create a new secret
resource "contabo_secret" "rootPassword" {
  name        = "my_secret"
	type        = "password"
	value 		  = "SmthSecure!1!!"
}

# Update an existing secret
resource "contabo_secret" "rootPassword" {
	value 		  = "MoreSecurePassword?1!"
}
