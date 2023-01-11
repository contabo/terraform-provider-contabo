package contabo

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// func TestAccContaboInstanceBasic(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccCheckInstanceDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: updateAndReinstallVPSCreation(),
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckContaboInstanceExists("contabo_instance.update_reinstall_test"),
// 					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "display_name", "created_display_name"),
// 				),
// 				PreventPostDestroyRefresh: true,
// 			},
// 			{
// 				Config: updateAndReinstallInstallFedora(),
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckContaboInstanceExists("contabo_instance.update_reinstall_test"),
// 					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "image_id", "1e1802ac-843c-42ed-9533-add37aaff46b"),
// 				),
// 				PreventPostDestroyRefresh: true,
// 			},
// 			{
// 				Config: updateAndReinstallDisplayNameUpdate(),
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckContaboInstanceExists("contabo_instance.update_reinstall_test"),
// 					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "display_name", "first_updated_display_name"),
// 				),
// 				PreventPostDestroyRefresh: true,
// 			},
// 			{
// 				Config: updateAndReinstallUpdateDisplayNameAndInstallArch(),
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckContaboInstanceExists("contabo_instance.update_reinstall_test"),
// 					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "display_name", "secound_updated_display_name"),
// 					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "image_id", "69b52ee3-2fda-4f44-b8de-69e480d87c7d"),
// 				),
// 				PreventPostDestroyRefresh: true,
// 			},
// 		},
// 	})
// }

func updateAndReinstallVPSCreation() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "update_reinstall_test" {
			display_name = "created_display_name"
			image_id = "66abf39a-ba8b-425e-a385-8eb347ceac10"
		}
	`
}

func updateAndReinstallDisplayNameUpdate() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "update_reinstall_test" {
			display_name = "first_updated_display_name"
		}
	`
}

func updateAndReinstallInstallFedora() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "update_reinstall_test" {
			image_id = "1e1802ac-843c-42ed-9533-add37aaff46b"
		}
	`
}

func updateAndReinstallUpdateDisplayNameAndInstallArch() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "update_reinstall_test" {
			image_id = "69b52ee3-2fda-4f44-b8de-69e480d87c7d"
			display_name = "secound_updated_display_name"
		}
	`
}

func testCheckContaboInstanceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No InstanceId set")
		}

		return nil
	}
}

func testAccCheckInstanceDestroy(s *terraform.State) error {
	return nil
}
