# Set a strong password for the VNC console of an instance. The API cannot
# turn the VNC console off, so a strong password is the best protection.
resource "random_password" "vnc" {
  length  = 16
  special = false
}

resource "contabo_instance_vnc" "database" {
  instance_id = contabo_instance.database.id
  password    = random_password.vnc.result
}
