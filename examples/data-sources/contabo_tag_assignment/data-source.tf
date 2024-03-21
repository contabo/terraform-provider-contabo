# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Get a specific tag assignment by tagId_resourceType_resourceId
data "contabo_tag_assignment" "default_tag_assignment" {
  id  = "178478_image_35ee288f-21ea-420c-a074-ce0a968b59c0"
}

output "output" {
  description = "output"
  value = data.contabo_tag_assignment.default_tag_assignment
}