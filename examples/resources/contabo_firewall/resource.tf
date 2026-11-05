# Allow SSH and HTTPS from everywhere and everything from the private
# network; the API drops all other inbound traffic.
resource "contabo_firewall" "web" {
  name         = "web"
  description  = "web servers"
  status       = "active"
  instance_ids = [contabo_instance.web.id]

  rules {
    inbound {
      display_name = "ssh"
      protocol     = "tcp"
      dest_ports   = ["22"]
      src_cidr {
        ipv4 = ["0.0.0.0/0"]
        ipv6 = ["::/0"]
      }
    }
    inbound {
      display_name = "https"
      protocol     = "tcp"
      dest_ports   = ["443"]
      src_cidr {
        ipv4 = ["0.0.0.0/0"]
        ipv6 = ["::/0"]
      }
    }
    inbound {
      display_name = "private network"
      protocol     = "any"
      src_cidr {
        ipv4 = ["10.0.0.0/8"]
      }
    }
  }
}
