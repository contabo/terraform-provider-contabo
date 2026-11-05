resource "contabo_private_network" "cluster" {
  name         = "cluster"
  region       = "EU"
  instance_ids = [contabo_instance.node.id]
}

# Instances need a reinstall to configure a newly attached private network.
# Reinstall whenever the private network changes.
resource "contabo_instance_reinstall" "node" {
  instance_id  = contabo_instance.node.id
  image_id     = data.contabo_image.ubuntu.id
  default_user = "root"
  ssh_keys     = [contabo_secret.ssh_key.id]
  user_data    = file("${path.module}/cloud-init.yaml")

  triggers = {
    private_network = contabo_private_network.cluster.id
  }
}

resource "contabo_instance" "node" {
  product_id = "V159"

  lifecycle {
    # contabo_instance_reinstall manages the installation.
    ignore_changes = [image_id, ssh_keys, root_password, user_data, default_user, application_id]
  }
}
