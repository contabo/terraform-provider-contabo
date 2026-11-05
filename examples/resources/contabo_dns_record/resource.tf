# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

resource "contabo_dns_zone" "example" {
  zone_name = "example.com"
}

# A record for www.example.com
resource "contabo_dns_record" "www" {
  zone_name = contabo_dns_zone.example.zone_name
  name      = "www"
  type      = "A"
  ttl       = 3600
  data      = "203.0.113.10"
}

# CNAME record pointing to www.example.com
resource "contabo_dns_record" "blog" {
  zone_name = contabo_dns_zone.example.zone_name
  name      = "blog"
  type      = "CNAME"
  data      = "www.example.com"
}

# MX record at the zone apex
resource "contabo_dns_record" "mx" {
  zone_name = contabo_dns_zone.example.zone_name
  name      = "@"
  type      = "MX"
  priority  = 10
  data      = "mail.example.com"
}

# TXT record at the zone apex
resource "contabo_dns_record" "spf" {
  zone_name = contabo_dns_zone.example.zone_name
  type      = "TXT"
  data      = "v=spf1 mx -all"
}

# SRV record
resource "contabo_dns_record" "sip" {
  zone_name = contabo_dns_zone.example.zone_name
  name      = "_sip._tcp"
  type      = "SRV"
  priority  = 10
  weight    = 5
  port      = 5060
  data      = "sip.example.com"
}

# CAA record
resource "contabo_dns_record" "caa" {
  zone_name = contabo_dns_zone.example.zone_name
  type      = "CAA"
  flag      = 0
  tag       = "issue"
  data      = "letsencrypt.org"
}
