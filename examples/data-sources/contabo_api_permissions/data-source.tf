# API names and actions that roles can grant for DNS.
data "contabo_api_permissions" "dns" {
  api_name_prefix = "/v1/dns"
}
