package contabo

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccObjectStorageBucketBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckObjectStorageBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: createBucketInEUObjectStorage(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboObjectStorageBucketExists("contabo_object_storage_bucket.my-lovely-bucket"),
					resource.TestCheckResourceAttr("contabo_object_storage_bucket.my-lovely-bucket", "name", "my-lovely-bucket"),
				),
				PreventPostDestroyRefresh: true,
				ExpectNonEmptyPlan:        true,
			},
		},
	})
}

func testAccCheckObjectStorageBucketDestroy(s *terraform.State) error {
	return nil
}

func createBucketInEUObjectStorage() string {
	return `
		provider "contabo" {}

		resource "contabo_object_storage" "new" {
			region                   = "EU"
			total_purchased_space_tb = 0.250
		}

		resource "contabo_object_storage_bucket" "my-lovely-bucket" {
			name = "my-lovely-bucket"
			object_storage_id = contabo_object_storage.new.id
		}	`
}

func testCheckContaboObjectStorageBucketExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ObjectStorageBucketId set")
		}
		time.Sleep(4 * time.Second)

		return nil
	}
}
