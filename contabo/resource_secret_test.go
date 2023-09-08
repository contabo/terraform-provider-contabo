package contabo

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"contabo.com/openapi"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var secretName = (uuid.New()).String()

func TestAccContaboSecretBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testCheckContaboSecretConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testCheckContaboSecretExists("contabo_secret.new"),
				),
			},
		},
	})
}

func testAccCheckSecretDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*openapi.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contabo_secret" {
			continue
		}

		id := rs.Primary.ID
		secretId, parseErr := strconv.ParseInt(id, 10, 64)

		if parseErr != nil {
			return parseErr
		}

		_, _, err := client.SecretsApi.
			RetrieveSecret(context.Background(), secretId).
			XRequestId((uuid.New()).String()).
			Execute()
		if err == nil {
			fmt.Printf("SECRET %v Still Exists: %v", secretId, err.Error())
			return nil
		}
	}

	return nil
}

func testCheckContaboSecretConfigBasic() string {
	return `
		provider "contabo" {}

		resource "contabo_secret" "new" {
		name        = "` + secretName + `"
		type        = "password"
		value 		= "AllCombinationPassword123?#"
		}
	`
}

func testCheckContaboSecretExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SecretId set")
		}

		return nil
	}
}
