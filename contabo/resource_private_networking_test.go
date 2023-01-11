package contabo

import (
	"context"
	"fmt"
	"strconv"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	uuid "github.com/satori/go.uuid"
)

// func TestAccContaboPrivateNetworkBasic(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck: func() {
// 			testAccPreCheck(t)
// 		},
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccCheckPrivateNetworkDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAddInstance(),
// 			},
// 			{
// 				Config: testCheckContaboPrivateNetworkConfigBasic(),
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckContaboPrivateNetworkExists("contabo_private_network.new"),
// 					resource.TestCheckResourceAttr("contabo_private_network.new", "instances.#", "0"),
// 				),
// 			},
// 			{
// 				Config: testContaboPrivateNetworkConfigWithInstance(),
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckContaboPrivateNetworkExists("contabo_private_network.with_instance"),
// 					resource.TestCheckResourceAttr("contabo_private_network.with_instance", "instances.#", "1"),
// 					resource.TestCheckResourceAttr(
// 						"contabo_private_network.with_instance", "instances.0.private_ip_config.0.v4.0.ip", "10.0.0.1"),
// 					resource.TestCheckResourceAttr("contabo_instance.new", "additional_ips.#", "0"),
// 				),
// 			},
// 		},
// 	})
// }

func testAccCheckPrivateNetworkDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*openapi.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contabo_private_network" {
			continue
		}

		id := rs.Primary.ID
		privateNetworktId, parseErr := strconv.ParseInt(id, 10, 64)

		if parseErr != nil {
			return parseErr
		}

		_, _, err := client.PrivateNetworksApi.
			RetrievePrivateNetwork(context.Background(), privateNetworktId).
			XRequestId(uuid.NewV4().String()).
			Execute()
		if err == nil {
			fmt.Printf("Private Network %v Still Exists: %v", privateNetworktId, err.Error())
			return nil
		}
	}

	return nil
}

func testAddInstance() string {
	return `
		resource "contabo_instance" "new" {
			display_name = "custom terraform"
		}
	`
}

func testCheckContaboPrivateNetworkConfigBasic() string {
	return `
		resource "contabo_private_network" "new" {
			name        = "terraform-test-private-network"
			description = "terraform test private network"
			region 		= "EU"
		}
	`
}

func testContaboPrivateNetworkConfigWithInstance() string {
	return `
		resource "contabo_instance" "new" {
			display_name = "custom terraform"
		}

		resource "contabo_private_network" "with_instance" {
			name			= "terraform-test-private-network-with-instance"
			region			= "EU"
			instance_ids 	= [
				contabo_instance.new.id
			]
		}
	`
}

func testCheckContaboPrivateNetworkExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No PrivateNetworkId set")
		}

		return nil
	}
}
