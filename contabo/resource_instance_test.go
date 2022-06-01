package contabo

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccContaboInstanceBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testCheckContaboInstanceConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboInstanceExists("contabo_instance.new"),
				),
			},
		},
	})
}

func testAccCheckInstanceDestroy(s *terraform.State) error {
	return nil
}

func testCheckContaboInstanceConfigBasic() string {
	return `
		provider "contabo" {}

		resource "contabo_instance" "new" {
			display_name = "custom terraform"
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
