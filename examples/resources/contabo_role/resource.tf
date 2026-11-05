# A role that can only manage DNS records.
resource "contabo_role" "dns" {
  name                 = "dns-records"
  access_all_resources = true

  permission {
    api_name = "/v1/dns/zones"
    actions  = ["READ"]
  }
  permission {
    api_name = "/v1/dns/zones/:zoneName/records"
    actions  = ["READ", "CREATE"]
  }
  permission {
    api_name = "/v1/dns/zones/:zoneName/records/:recordId"
    actions  = ["UPDATE", "DELETE"]
  }
}

resource "contabo_tag" "team_a" {
  name  = "team-a"
  color = "#0A78C3"
}

# A role limited to the instances that carry a tag.
resource "contabo_role" "team_a_instances" {
  name = "team-a-instances"

  permission {
    api_name  = "/v1/compute/instances"
    actions   = ["READ"]
    resources = [contabo_tag.team_a.id]
  }
}
