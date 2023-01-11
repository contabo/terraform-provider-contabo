package contabo

import (
	"context"
	"fmt"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	uuid "github.com/satori/go.uuid"
)

// func TestAccContaboFirewallBasic(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccCheckFirewallDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testCheckContaboFirewallConfigBasic(),
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckContaboFirewallExists("contabo_firewall.new"),
// 				),
// 				ExpectNonEmptyPlan: true,
// 			},
// 		},
// 	})
// }

func testAccCheckFirewallDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*openapi.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contabo_firewall" {
			continue
		}

		firewallId := rs.Primary.ID

		_, _, err := client.FirewallsApi.
			RetrieveFirewall(context.Background(), firewallId).
			XRequestId(uuid.NewV4().String()).
			Execute()
		if err == nil {
			fmt.Printf("Firewall %v Still Exists: %v", firewallId, err.Error())
			return nil
		}
	}

	return nil
}

func testCheckContaboFirewallConfigBasic() string {
	return `
		provider "contabo" {}

		resource "contabo_firewall" "new" {
		name        = "terraform-firewall"
		description	= "terraform-description"
		status 		= "active"
		rules {
			inbound {
				protocol = "tcp"
				action = "accept"
				status = "active"
				dest_ports = ["666"]
				src_cidr {
						ipv4 = ["127.0.0.1", "6.6.6.6"]
					}
				}
			}
		}
	`
}

func testCheckContaboFirewallExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No FirewallId set")
		}

		return nil
	}
}
