# Configure your Contabo API credentials in provider stanza
provider "contabo" {
  oauth2_client_id = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user = "[your username]"
  oauth2_pass = "[your password]"
}

# Create custom image with apline iso
resource "contabo_image" "custom_image_alpine" {
  name = "custom_alpine"
  image_url = "https://dl-cdn.alpinelinux.org/alpine/v3.13/releases/s390x/alpine-standard-3.13.5-s390x.iso"
  os_type = "Linux"
  version = "v3.13.5"
  description = "custom alpine iso image"
}

# Update name of custom image
resource "contabo_image" "custom_image_alpine" {
  name = "custom_alpine_v3.13.5"
}
