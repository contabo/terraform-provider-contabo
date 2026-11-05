data "contabo_instance_vnc" "database" {
  instance_id = contabo_instance.database.id
}

output "vnc" {
  value = "${data.contabo_instance_vnc.database.vnc_ip}:${data.contabo_instance_vnc.database.vnc_port}"
}
