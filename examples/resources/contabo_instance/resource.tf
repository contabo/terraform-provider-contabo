# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Create a new object storage in region EU
resource "contabo_instance" "database_instance" {
  name       = "database"
  product_id = "V2"
  region     = "EU"
  period     = 3 
}

# Update custom image on instance
resource "contabo_instance" "database_instance" {
  image_id = contabo_image.custom_image_alpine.id
}
