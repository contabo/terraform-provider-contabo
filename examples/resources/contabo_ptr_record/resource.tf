# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

data "contabo_instance" "mail" {
  id = "12345"
}

# Reverse DNS for the IPv4 address of an instance.
# IPv4 PTR records always exist: this updates the record, and destroying the
# resource leaves the current value in place.
resource "contabo_ptr_record" "mail_ipv4" {
  ip_address = data.contabo_instance.mail.ip_config[0].v4[0].ip
  ptr        = "mail.example.com"
}

# Reverse DNS for an IPv6 address of an instance.
# IPv6 PTR records are created and deleted with the resource.
resource "contabo_ptr_record" "mail_ipv6" {
  ip_address = "2001:db8:1234:5678::25"
  ptr        = "mail.example.com"
  ttl        = 3600
}
