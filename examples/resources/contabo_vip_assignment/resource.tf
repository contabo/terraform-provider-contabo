# Assign a floating IP, bought as an add-on, to the API server.
data "contabo_vips" "floating" {
  type = "floating"
}

resource "contabo_vip_assignment" "api" {
  ip          = data.contabo_vips.floating.ips[0]
  resource_id = contabo_instance.control_plane.id
}
