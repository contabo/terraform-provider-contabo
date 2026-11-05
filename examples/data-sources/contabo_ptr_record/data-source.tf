# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Get the PTR record of an IP address
data "contabo_ptr_record" "example" {
  ip_address = "203.0.113.10"
}

output "ptr" {
  value = data.contabo_ptr_record.example.ptr
}
