# Order a Cloud VPS in the EU region with a 3 month contract, private
# networking and cloud-init.
resource "contabo_instance" "database" {
  display_name = "database"
  product_id   = "V153"
  region       = "EU"
  period       = 3
  image_id     = data.contabo_image.ubuntu.id
  default_user = "root"
  ssh_keys     = [contabo_secret.ssh_key.id]
  user_data    = file("${path.module}/cloud-init.yaml")

  add_ons {
    id = "1477" # private networking
  }
}

# Manage an instance that was ordered in the customer control panel without
# reinstalling it, and keep it when the resource is destroyed.
resource "contabo_instance" "existing" {
  existing_instance_id = "203235629"
  display_name         = "legacy"
  cancel_on_destroy    = false
}
