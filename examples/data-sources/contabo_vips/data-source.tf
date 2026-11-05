# Floating IPs that are not assigned yet.
data "contabo_vips" "free" {
  type       = "floating"
  unassigned = true
}
