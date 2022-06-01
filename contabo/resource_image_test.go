package contabo

import (
	"context"
	"fmt"
	"testing"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	uuid "github.com/satori/go.uuid"
)

func TestAccContaboImageBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: testCheckContaboImageConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboImageExists("contabo_image.new"),
				),
			},
		},
	})
}

func testAccCheckImageDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*openapi.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contabo_image" {
			continue
		}

		imageId := rs.Primary.ID

		_, _, err := client.ImagesApi.
			RetrieveImage(context.Background(), imageId).
			XRequestId(uuid.NewV4().String()).
			Execute()
		if err == nil {
			fmt.Printf("IMAGE %v Still Exists: %v", imageId, err.Error())
			return nil
		}
	}

	return nil
}

func testCheckContaboImageConfigBasic() string {
	return `
		provider "contabo" {}

		resource "contabo_image" "new" {
		name        = "custom_alpi"
		image_url   = "https://dl-cdn.alpinelinux.org/alpine/v3.13/releases/s390x/alpine-standard-3.13.5-s390x.iso"
		os_type     = "Linux"
		version     = "0.0.1"
		description = "custom alpi"
		}
	`
}

func testCheckContaboImageExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ImageId set")
		}

		return nil
	}
}
