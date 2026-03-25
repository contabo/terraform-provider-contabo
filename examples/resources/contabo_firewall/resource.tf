# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user = "[your username]"
  oauth2_pass = "[your password]"
}

# Create a default firewall
resource "contabo_firewall" "default" {
name        = "default"
description	= "default firewall"
status 		  = "active"
rules {
	inbound {
		protocol   = "tcp"
		action     = "accept"
		status     = "active"
		dest_ports = ["22", "80", "443"]
		src_cidr {
				ipv4 = ["194.165.134.20", "194.165.134.21"]
				ipv6 = ["2001:0db8:85a3:0000:0000:8a2e:0370:7334"]
			}
		}
	}
}

# Update a the default firewall
resource "contabo_firewall" "default" {
name        = "default"
description	= "default firewall"
status 		  = "active"
rules {
	inbound {
		protocol   = "tcp"
		action     = "accept"
		status     = "active"
		dest_ports = ["22", "80", "443", "25"]
		src_cidr {
				ipv4   = ["194.165.134.20", "194.165.134.21", "194.165.134.22", "194.165.134.23"]
				ipv6   = ["2001:0db8:85a3:0000:0000:8a2e:0370:7334"]
			}
		}
	}
}
