# All running Cloud VPS Plus instances whose display name starts with "k8s-".
data "contabo_instances" "cluster" {
  display_name_prefix = "k8s-"
  product_ids         = ["V159", "V160"]
  statuses            = ["running"]
}

output "cluster_ips" {
  value = [for instance in data.contabo_instances.cluster.instances : instance.ip_config[0].v4[0].ip]
}
