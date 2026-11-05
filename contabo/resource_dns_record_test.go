package contabo

import (
	"context"
	"fmt"
	"os"
	"testing"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	uuid "github.com/satori/go.uuid"
)

// The DNS acceptance tests need a zone name that is not yet hosted on the
// Contabo name servers, e.g. CNTB_TEST_DNS_ZONE=example.com. The zone is
// created and deleted by the test.
func testAccDnsZoneName(t *testing.T) string {
	zoneName := os.Getenv("CNTB_TEST_DNS_ZONE")
	if zoneName == "" {
		t.Skip("CNTB_TEST_DNS_ZONE must be set for DNS acceptance tests")
	}
	return zoneName
}

func TestAccContaboDnsZoneAndRecords(t *testing.T) {
	zoneName := testAccDnsZoneName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDnsZoneAndRecordsConfig(zoneName, 300, "1.1.1.1", "v=spf1 -all"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_dns_zone.test", "id", zoneName),
					resource.TestCheckResourceAttr("contabo_dns_zone.test", "name_servers.#", "3"),
					resource.TestCheckResourceAttr("data.contabo_dns_zone.test", "name_servers.#", "3"),
					testAccCheckDnsRecordExists("contabo_dns_record.a"),
					resource.TestCheckResourceAttr("contabo_dns_record.a", "name", "www"),
					resource.TestCheckResourceAttr("contabo_dns_record.a", "fqdn", "www."+zoneName),
					resource.TestCheckResourceAttr("contabo_dns_record.a", "ttl", "300"),
					resource.TestCheckResourceAttr("contabo_dns_record.a", "data", "1.1.1.1"),
					testAccCheckDnsRecordExists("contabo_dns_record.aaaa"),
					testAccCheckDnsRecordExists("contabo_dns_record.cname"),
					resource.TestCheckResourceAttr("contabo_dns_record.cname", "data", "www."+zoneName),
					testAccCheckDnsRecordExists("contabo_dns_record.txt"),
					resource.TestCheckResourceAttr("contabo_dns_record.txt", "name", ""),
					resource.TestCheckResourceAttr("contabo_dns_record.txt", "data", "v=spf1 -all"),
					testAccCheckDnsRecordExists("contabo_dns_record.mx"),
					resource.TestCheckResourceAttr("contabo_dns_record.mx", "priority", "10"),
					testAccCheckDnsRecordExists("contabo_dns_record.srv"),
					resource.TestCheckResourceAttr("contabo_dns_record.srv", "port", "5060"),
					resource.TestCheckResourceAttr("contabo_dns_record.srv", "weight", "5"),
					testAccCheckDnsRecordExists("contabo_dns_record.caa"),
					resource.TestCheckResourceAttr("contabo_dns_record.caa", "tag", "issue"),
				),
			},
			{
				Config: testAccDnsZoneAndRecordsConfig(zoneName, 600, "1.0.0.1", "v=spf1 mx -all"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_dns_record.a", "ttl", "600"),
					resource.TestCheckResourceAttr("contabo_dns_record.a", "data", "1.0.0.1"),
					resource.TestCheckResourceAttr("contabo_dns_record.txt", "data", "v=spf1 mx -all"),
				),
			},
			{
				ResourceName:      "contabo_dns_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "contabo_dns_record.a",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "contabo_dns_record.srv",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "contabo_dns_record.caa",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDnsZoneAndRecordsConfig(zoneName string, ttl int, ipv4 string, txt string) string {
	return fmt.Sprintf(`
provider "contabo" {}

resource "contabo_dns_zone" "test" {
  zone_name = %[1]q
}

data "contabo_dns_zone" "test" {
  zone_name = contabo_dns_zone.test.zone_name
}

resource "contabo_dns_record" "a" {
  zone_name = contabo_dns_zone.test.zone_name
  name      = "www"
  type      = "A"
  ttl       = %[2]d
  data      = %[3]q
}

resource "contabo_dns_record" "aaaa" {
  zone_name = contabo_dns_zone.test.zone_name
  name      = "www.%[1]s"
  type      = "AAAA"
  data      = "2606:4700:4700:0000:0000:0000:0000:1111"
}

resource "contabo_dns_record" "cname" {
  zone_name = contabo_dns_zone.test.zone_name
  name      = "alias"
  type      = "CNAME"
  data      = "www.%[1]s."
}

resource "contabo_dns_record" "txt" {
  zone_name = contabo_dns_zone.test.zone_name
  name      = "@"
  type      = "TXT"
  data      = "\"%[4]s\""
}

resource "contabo_dns_record" "mx" {
  zone_name = contabo_dns_zone.test.zone_name
  type      = "MX"
  priority  = 10
  data      = "mail.%[1]s"
}

resource "contabo_dns_record" "srv" {
  zone_name = contabo_dns_zone.test.zone_name
  name      = "_sip._tcp"
  type      = "SRV"
  priority  = 10
  weight    = 5
  port      = 5060
  data      = "sip.%[1]s"
}

resource "contabo_dns_record" "caa" {
  zone_name = contabo_dns_zone.test.zone_name
  type      = "CAA"
  flag      = 0
  tag       = "issue"
  data      = "letsencrypt.org"
}
`, zoneName, ttl, ipv4, txt)
}

func testAccCheckDnsRecordExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		zoneName, recordId, err := parseDnsRecordId(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := testAccProvider.Meta().(*openapi.APIClient)
		records, _, err := listDnsZoneRecords(context.Background(), client, zoneName)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record.RecordId == recordId {
				return nil
			}
		}
		return fmt.Errorf("DNS record %s not found", rs.Primary.ID)
	}
}

func testAccCheckDnsZoneDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*openapi.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contabo_dns_zone" {
			continue
		}

		httpResp, err := client.DNSApi.
			RetrieveDnsZone(context.Background(), rs.Primary.ID).
			XRequestId(uuid.NewV4().String()).
			Execute()
		if err == nil {
			return fmt.Errorf("DNS zone %s still exists", rs.Primary.ID)
		}
		if !isNotFound(httpResp) {
			return err
		}
	}

	return nil
}
