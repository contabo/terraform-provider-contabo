terraform {
  required_providers {
    contabo = {
      source = "contabo/contabo"
      version = ">= 0.1.17"
    }
  }
}

# Configure your Contabo API credentials in provider stanza
provider "contabo" {
  oauth2_client_id = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user = "[your username]"
  oauth2_pass = "[your password]"
}


# Set a default image so we can access it by name
data "contabo_image" "debian_11" {
  id = "66abf39a-ba8b-425e-a385-8eb347ceac10"
}

# or specify one with our custom image
resource "contabo_instance" "custom_instance" {
  display_name = "Debian 11 instance"
  image_id = contabo_image.debian_11.id
}

# Output our newly created instances
output "custom_instance_output" {
  description = "Our first custom instance"
  value = contabo_instance.custom_instance
}
