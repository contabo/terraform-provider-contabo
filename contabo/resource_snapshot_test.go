package contabo

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	uuid "github.com/satori/go.uuid"
)

var instanceId int64 = 10001001

func TestAccContaboSnapshotBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceSnapshotDestroy,
		Steps: []resource.TestStep{
			{
				Config: testCheckContaboInstanceSnapshotConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboInstanceSnapshotExists("contabo_instance_snapshot.new"),
				),
			},
		},
	})

}

func testAccCheckInstanceSnapshotDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*openapi.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contabo_instance_snapshot" {
			continue
		}

		id := rs.Primary.ID

		_, _, err := client.SnapshotsApi.
			RetrieveSnapshot(context.Background(), instanceId, id).
			XRequestId(uuid.NewV4().String()).
			Execute()
		if err == nil {
			fmt.Printf("SNAPSHOT %v Still Exists: %v", id, err)
			return nil
		}
	}

	return nil
}

func testCheckContaboInstanceSnapshotConfigBasic() string {
	return `
			provider "contabo" {}
	
			resource "contabo_instance_snapshot" "new" {
				name = "test-snapshot"
				description = "terraform test-snapshot"
				instance_id = ` + strconv.FormatInt(instanceId, 10) + `
				}
		`
}

func testCheckContaboInstanceSnapshotExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Snapshot set")
		}

		return nil
	}
}
