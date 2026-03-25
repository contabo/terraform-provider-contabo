package contabo

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const resourceName = "contabo_firewall.import"

func TestContaboFirewallImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFirewallDestroy,
		Steps: []resource.TestStep{
			{
				Config:             testCheckContaboFirewallConfigImport(),
				ExpectNonEmptyPlan: true,
			},
			{
				ResourceName:       resourceName,
				ImportState:        true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckContaboFirewallConfigImport() string {
	return `
		provider "contabo" {}

		resource "contabo_firewall" "import" {
		name        = "terraform-firewall-import"
		description	= "terraform-description-import"
		status 		= "active"
		rules {
			inbound {
				protocol   = "tcp"
				action     = "accept"
				status     = "active"
				dest_ports = ["22", "80", "443"]
				src_cidr {
						ipv4 = ["194.165.134.20", "194.165.134.21"]
					}
				}
			}
		}
	`
}
