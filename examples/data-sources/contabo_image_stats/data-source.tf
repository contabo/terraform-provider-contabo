data "contabo_image_stats" "current" {}

resource "contabo_image" "talos" {
  name      = "talos"
  image_url = "https://example.com/talos-nocloud-amd64.qcow2"
  os_type   = "Linux"
  version   = "1.11"

  lifecycle {
    precondition {
      condition     = data.contabo_image_stats.current.free_size_mb > 1024
      error_message = "Custom image storage is full or missing; order the custom image storage add-on first."
    }
  }
}
