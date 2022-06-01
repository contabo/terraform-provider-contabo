package contabo

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"contabo": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if err := os.Getenv("CNTB_API"); err == "" {
		t.Fatal("CNTB_API must be set")
	}
	if err := os.Getenv("CNTB_OAUTH2_TOKEN_URL"); err == "" {
		t.Fatal("CNTB_OAUTH2_TOKEN_URL must be set")
	}
	if err := os.Getenv("CNTB_OAUTH2_CLIENT_ID"); err == "" {
		t.Fatal("CNTB_OAUTH2_CLIENT_ID must be set")
	}
	if err := os.Getenv("CNTB_OAUTH2_CLIENT_SECRET"); err == "" {
		t.Fatal("CNTB_OAUTH2_CLIENT_SECRET must be set")
	}
	if err := os.Getenv("CNTB_OAUTH2_USER"); err == "" {
		t.Fatal("CNTB_OAUTH2_USER must be set")
	}
	if err := os.Getenv("CNTB_OAUTH2_PASS"); err == "" {
		t.Fatal("CNTB_OAUTH2_PASS must be set")
	}
}
