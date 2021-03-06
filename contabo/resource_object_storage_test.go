package contabo

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// func TestAccContaboObjectStorageBasic(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccCheckObjectStorageDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testCheckContaboObjectStorageConfigBasic(),
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckContaboObjectStorageExists("contabo_object_storage.object_storage_eu"),
// 				),
// 				// ToDo: Object storage plan is not empty
// 				ExpectNonEmptyPlan: true,
// 			},
// 		},
// 	})
// }

func testAccCheckObjectStorageDestroy(s *terraform.State) error {
	return nil
}

func testCheckContaboObjectStorageConfigBasic() string {
	return `
		provider "contabo" {}

		resource "contabo_object_storage" "object_storage_eu" {
			region                   = "EU"
			total_purchased_space_tb = 2
		}
	`
}

func testCheckContaboObjectStorageExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ObjectStorageId set")
		}
		time.Sleep(4 * time.Second)

		return nil
	}
}
