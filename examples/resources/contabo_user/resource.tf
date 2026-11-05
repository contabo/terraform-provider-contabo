# A user for an external-dns or cert-manager webhook. The API emails the
# user a link to set a password.
resource "contabo_user" "dns_bot" {
  email      = "dns-bot@example.com"
  first_name = "DNS"
  last_name  = "Bot"
  roles      = [contabo_role.dns.id]
}
