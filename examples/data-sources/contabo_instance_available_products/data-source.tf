data "contabo_instance_available_products" "upgrades" {
  instance_id = contabo_instance.database.id
}

output "upgrade_options" {
  value = [for p in data.contabo_instance_available_products.upgrades.products : "${p.product_id}: ${p.name} (+${p.price_difference})"]
}
