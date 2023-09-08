package contabo

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var creationDisplayName = (uuid.New()).String()
var updatedDisplayName = (uuid.New()).String()
var anotherUpdatedDisplayName = (uuid.New()).String()

func TestAccContaboInstanceBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: updateAndReinstallVPSCreation(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboInstanceExists("contabo_instance.update_reinstall_test"),
					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "display_name", creationDisplayName),
				),
				PreventPostDestroyRefresh: true,
			},
			{
				Config: updateAndReinstallInstallFedora(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboInstanceExists("contabo_instance.update_reinstall_test"),
					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "image_id", "66abf39a-ba8b-425e-a385-8eb347ceac10"),
				),
				PreventPostDestroyRefresh: true,
			},
			{
				Config: updateAndReinstallDisplayNameUpdate(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboInstanceExists("contabo_instance.update_reinstall_test"),
					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "display_name", updatedDisplayName),
				),
				PreventPostDestroyRefresh: true,
			},
			{
				Config: updateAndReinstallUpdateDisplayNameAndInstallArch(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboInstanceExists("contabo_instance.update_reinstall_test"),
					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "display_name", anotherUpdatedDisplayName),
					resource.TestCheckResourceAttr("contabo_instance.update_reinstall_test", "image_id", "66abf39a-ba8b-425e-a385-8eb347ceac10"),
				),
				PreventPostDestroyRefresh: true,
			},
		},
	})
}

func updateAndReinstallVPSCreation() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "update_reinstall_test" {
			display_name = "` + creationDisplayName + `"
			image_id = "66abf39a-ba8b-425e-a385-8eb347ceac10"
		}
	`
}

func updateAndReinstallDisplayNameUpdate() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "update_reinstall_test" {
			display_name = "` + updatedDisplayName + `"
		}
	`
}

func updateAndReinstallInstallFedora() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "update_reinstall_test" {
			image_id = "66abf39a-ba8b-425e-a385-8eb347ceac10"
		}
	`
}

func updateAndReinstallUpdateDisplayNameAndInstallArch() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "update_reinstall_test" {
			image_id = "66abf39a-ba8b-425e-a385-8eb347ceac10"
			display_name = "` + anotherUpdatedDisplayName + `"
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
