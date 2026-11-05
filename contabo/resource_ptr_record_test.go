package contabo

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"testing"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	uuid "github.com/satori/go.uuid"
)

// CNTB_TEST_PTR_IPV6 must be an unused IPv6 address of one of your instances,
// e.g. <instance /64 prefix>::100. A PTR record is created and deleted for it.
func TestAccContaboPtrRecordIPv6(t *testing.T) {
	ipAddress := os.Getenv("CNTB_TEST_PTR_IPV6")
	if ipAddress == "" {
		t.Skip("CNTB_TEST_PTR_IPV6 must be set for IPv6 PTR acceptance tests")
	}
	canonicalIp := net.ParseIP(ipAddress).String()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPtrRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPtrRecordConfig(ipAddress, "ptr-test-1.example.com.", "ttl = 3600"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_ptr_record.test", "id", canonicalIp),
					resource.TestCheckResourceAttr("contabo_ptr_record.test", "ptr", "ptr-test-1.example.com"),
					resource.TestCheckResourceAttr("contabo_ptr_record.test", "ttl", "3600"),
					resource.TestCheckResourceAttr("data.contabo_ptr_record.test", "ptr", "ptr-test-1.example.com"),
				),
			},
			{
				Config: testAccPtrRecordConfig(ipAddress, "ptr-test-2.example.com", "ttl = 3600"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_ptr_record.test", "ptr", "ptr-test-2.example.com"),
				),
			},
			{
				ResourceName:      "contabo_ptr_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// CNTB_TEST_PTR_IPV4 must be an IPv4 address of one of your instances. Its PTR
// record is changed during the test and restored to the original value
// afterwards.
func TestAccContaboPtrRecordIPv4(t *testing.T) {
	ipAddress := os.Getenv("CNTB_TEST_PTR_IPV4")
	if ipAddress == "" {
		t.Skip("CNTB_TEST_PTR_IPV4 must be set for IPv4 PTR acceptance tests")
	}
	var originalPtr string

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			// IPv4 PTR records are not deleted on destroy, restore the original value.
			if originalPtr == "" {
				return nil
			}
			client := testAccProvider.Meta().(*openapi.APIClient)
			if diags := updatePtrRecord(context.Background(), client, ipAddress, originalPtr, nil); diags.HasError() {
				return fmt.Errorf("restoring PTR record of %s: %v", ipAddress, diags)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "contabo" {}

data "contabo_ptr_record" "original" {
  ip_address = %q
}
`, ipAddress),
				Check: func(s *terraform.State) error {
					originalPtr = s.RootModule().Resources["data.contabo_ptr_record.original"].Primary.Attributes["ptr"]
					if originalPtr == "" {
						return fmt.Errorf("no PTR record found for %s", ipAddress)
					}
					return nil
				},
			},
			{
				Config: testAccPtrRecordConfig(ipAddress, "ptr-test-ipv4.example.com", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_ptr_record.test", "ptr", "ptr-test-ipv4.example.com"),
					resource.TestCheckResourceAttr("contabo_ptr_record.test", "ttl", "86400"),
				),
			},
			{
				Config:      testAccPtrRecordConfig(ipAddress, "ptr-test-ipv4.example.com", "ttl = 300"),
				ExpectError: regexp.MustCompile("ttl can only be set for IPv6 PTR records"),
			},
		},
	})
}

func testAccPtrRecordConfig(ipAddress string, ptr string, extra string) string {
	return fmt.Sprintf(`
provider "contabo" {}

resource "contabo_ptr_record" "test" {
  ip_address = %q
  ptr        = %q
  %s
}

data "contabo_ptr_record" "test" {
  ip_address = contabo_ptr_record.test.ip_address
}
`, ipAddress, ptr, extra)
}

func testAccCheckPtrRecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*openapi.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contabo_ptr_record" {
			continue
		}

		_, httpResp, err := client.DNSApi.
			RetrievePtrRecord(context.Background(), rs.Primary.ID).
			XRequestId(uuid.NewV4().String()).
			Execute()
		if err == nil {
			return fmt.Errorf("PTR record for %s still exists", rs.Primary.ID)
		}
		if !isNotFound(httpResp) {
			return err
		}
	}

	return nil
}
