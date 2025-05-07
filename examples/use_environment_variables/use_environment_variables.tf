terraform {
  required_providers {
    contabo = {
      source = "contabo/contabo"
      version = ">= 0.1.32"
    }
  }
}

provider "contabo" {
  # Set the following environment variables:
  #
  # CNTB_OAUTH2_CLIENT_ID
  # CNTB_OAUTH2_CLIENT_SECRET
  # CNTB_OAUTH2_USER
  # CNTB_OAUTH2_PASS
  #
  # and you are good to go
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
  value = contabo_instance.default_instance
}
