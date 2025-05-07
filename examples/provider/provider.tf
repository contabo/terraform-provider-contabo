terraform {
  required_providers {
    contabo = {
      source = "contabo/contabo"
      version = ">= 0.1.32"
    }
  }
}

# Configure your Contabo API credentials in provider stanza
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Create a default contabo VPS instance
resource "contabo_instance" "default_instance" {}

# Output our newly created instances
output "default_instance_output" {
  description = "Our first default instance"
  value       = contabo_instance.default_instance
}
