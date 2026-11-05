data "contabo_instance" "database" {
  id = "203235629"
}

output "database_ip" {
  value = data.contabo_instance.database.ip_config[0].v4[0].ip
}
