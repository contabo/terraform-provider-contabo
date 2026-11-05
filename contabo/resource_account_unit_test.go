package contabo

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestUnitVipAssignment(t *testing.T) {
	api := newFakeAPI(t)
	instanceId := api.addInstance(nil)
	otherId := api.addInstance(nil)
	api.addVip("198.51.100.20", "floating")
	api.addVip("198.51.100.21", "additional")

	config := func(target int64) string {
		return api.providerConfig() + fmt.Sprintf(`
resource "contabo_vip_assignment" "api" {
  ip          = "198.51.100.20"
  resource_id = "%d"
}

data "contabo_vips" "free" {
  unassigned = true
  depends_on = [contabo_vip_assignment.api]
}

data "contabo_vip" "api" {
  ip         = contabo_vip_assignment.api.ip
  depends_on = [contabo_vip_assignment.api]
}
`, target)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config(instanceId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_vip_assignment.api", "id", "198.51.100.20"),
					resource.TestCheckResourceAttr("contabo_vip_assignment.api", "resource_type", "instances"),
					resource.TestCheckResourceAttr("contabo_vip_assignment.api", "type", "floating"),
					resource.TestCheckResourceAttr("data.contabo_vips.free", "ips.#", "1"),
					resource.TestCheckResourceAttr("data.contabo_vips.free", "ips.0", "198.51.100.21"),
					resource.TestCheckResourceAttr("data.contabo_vip.api", "resource_id", fmt.Sprint(instanceId)),
				),
			},
			{
				// Moving the VIP replaces the assignment.
				Config: config(otherId),
				Check:  resource.TestCheckResourceAttr("contabo_vip_assignment.api", "resource_id", fmt.Sprint(otherId)),
			},
			{
				// The import steps use a configuration without data sources,
				// which older SDK versions cannot match during the import check.
				Config: api.providerConfig() + fmt.Sprintf(`
resource "contabo_vip_assignment" "api" {
  ip          = "198.51.100.20"
  resource_id = "%d"
}
`, otherId),
				ResourceName:      "contabo_vip_assignment.api",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			if api.vips["198.51.100.20"]["resourceId"] != "" {
				return fmt.Errorf("VIP still assigned")
			}
			return nil
		},
	})
}

func TestUnitRoleAndUser(t *testing.T) {
	api := newFakeAPI(t)

	config := func(actions string, firstName string) string {
		return api.providerConfig() + fmt.Sprintf(`
resource "contabo_role" "dns" {
  name = "dns-editor"
  permission {
    api_name  = "/v1/dns/zones"
    actions   = [%s]
    resources = [42]
  }
  permission {
    api_name = "/v1/dns/zones/:zoneName/records"
    actions  = ["READ", "CREATE", "UPDATE", "DELETE"]
  }
}

resource "contabo_user" "dns" {
  email      = "dns-bot@example.com"
  first_name = %q
  last_name  = "Bot"
  roles      = [contabo_role.dns.id]
}

data "contabo_role" "dns" {
  name       = contabo_role.dns.name
  depends_on = [contabo_role.dns]
}

data "contabo_api_permissions" "dns" {
  api_name_prefix = "/v1/dns"
}

data "contabo_users" "all" {
  depends_on = [contabo_user.dns]
}
`, actions, firstName)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config(`"READ"`, "Dns"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_role.dns", "type", "custom"),
					resource.TestCheckResourceAttr("contabo_role.dns", "permission.#", "2"),
					resource.TestCheckResourceAttr("contabo_user.dns", "roles.#", "1"),
					resource.TestCheckResourceAttr("contabo_user.dns", "locale", "en"),
					resource.TestCheckResourceAttr("data.contabo_role.dns", "permission.#", "2"),
					resource.TestCheckResourceAttr("data.contabo_api_permissions.dns", "permissions.#", "2"),
					resource.TestCheckResourceAttr("data.contabo_users.all", "users.#", "1"),
				),
			},
			{
				Config: config(`"READ", "UPDATE"`, "Dns2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_user.dns", "first_name", "Dns2"),
				),
			},
			{
				// The import steps use a configuration without data sources,
				// which older SDK versions cannot match during the import check.
				Config:            resourcesOnly(config(`"READ", "UPDATE"`, "Dns2")),
				ResourceName:      "contabo_role.dns",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:            resourcesOnly(config(`"READ", "UPDATE"`, "Dns2")),
				ResourceName:      "contabo_user.dns",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			if len(api.roles) != 0 || len(api.users) != 0 {
				return fmt.Errorf("role or user not deleted")
			}
			return nil
		},
	})
}

func TestUnitAccountDataSources(t *testing.T) {
	api := newFakeAPI(t)
	instanceId := api.addInstance(nil)
	accessKey := api.addCredential("os-1", "European Union")
	api.addCredential("os-2", "United States (Central)")
	api.products = []map[string]interface{}{
		{"productId": "V154", "name": "Cloud VPS 6", "frontendName": "Cloud VPS 6", "category": "ssd", "ramSizeGb": 12, "diskSizeGb": 200, "cpuCores": 6, "netSpeed": 300, "snapshots": 2, "vsPrice": 9, "current": false, "locationChangeRequired": false, "liveMigrationAvailable": false, "liveMigrationDisabledReason": "category_change", "storageExtensionAvailable": true, "backupAvailable": true, "priceDifferenceWithStorageExtension": 46.08, "priceDifferenceWithoutStorageExtension": 26.88},
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: api.providerConfig() + fmt.Sprintf(`
data "contabo_object_storage_credentials" "eu" {
  object_storage_id = "os-1"
}
data "contabo_object_storage_credentials" "all" {}
data "contabo_image_stats" "stats" {}
data "contabo_instance_available_products" "upgrades" {
  instance_id = "%d"
}
data "contabo_instance_vnc" "vnc" {
  instance_id = "%d"
}
`, instanceId, instanceId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.contabo_object_storage_credentials.eu", "access_key", accessKey),
					resource.TestCheckResourceAttr("data.contabo_object_storage_credentials.eu", "credentials.#", "1"),
					resource.TestCheckResourceAttr("data.contabo_object_storage_credentials.all", "credentials.#", "2"),
					resource.TestCheckResourceAttr("data.contabo_object_storage_credentials.all", "user_id", api.userId),
					resource.TestCheckResourceAttr("data.contabo_image_stats.stats", "free_size_mb", "24576"),
					resource.TestCheckResourceAttr("data.contabo_instance_available_products.upgrades", "products.0.product_id", "V154"),
					resource.TestCheckResourceAttr("data.contabo_instance_available_products.upgrades", "products.0.price_difference", "26.88"),
					resource.TestCheckResourceAttr("data.contabo_instance_vnc.vnc", "vnc_port", "63031"),
				),
			},
			{
				Config: api.providerConfig() + `
data "contabo_object_storage_credentials" "none" {
  object_storage_id = "missing"
}
`,
				ExpectError: regexp.MustCompile(`no object storage credentials`),
			},
		},
	})
}

func TestUnitInstanceReinstallResource(t *testing.T) {
	api := newFakeAPI(t)
	instanceId := api.addInstance(map[string]interface{}{"defaultUser": "root"})

	config := func(trigger string) string {
		return api.providerConfig() + fmt.Sprintf(`
resource "contabo_instance_reinstall" "talos" {
  instance_id  = "%d"
  image_id     = "talos-image"
  default_user = "root"
  user_data    = "machine: {}"
  triggers = {
    private_network = %q
  }
}
`, instanceId, trigger)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("pn-1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_instance_reinstall.talos", "status", "running"),
					func(s *terraform.State) error {
						if len(api.reinstalls) != 1 || api.reinstalls[0]["imageId"] != "talos-image" || api.reinstalls[0]["userData"] != "machine: {}" {
							return fmt.Errorf("unexpected reinstalls: %v", api.reinstalls)
						}
						return nil
					},
				),
			},
			{
				Config: config("pn-1"),
				Check: func(s *terraform.State) error {
					if len(api.reinstalls) != 1 {
						return fmt.Errorf("reinstalled without a change")
					}
					return nil
				},
			},
			{
				Config: config("pn-2"),
				Check: func(s *terraform.State) error {
					if len(api.reinstalls) != 2 {
						return fmt.Errorf("trigger change did not reinstall")
					}
					return nil
				},
			},
		},
	})
}

func TestUnitInstanceVncPassword(t *testing.T) {
	api := newFakeAPI(t)
	instanceId := api.addInstance(nil)

	config := func(password string) string {
		return api.providerConfig() + fmt.Sprintf(`
resource "contabo_instance_vnc" "vnc" {
  instance_id = "%d"
  password    = %q
}
`, instanceId, password)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config("short"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`at least 8 characters`),
			},
			{
				Config: config("Secret123"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_instance_vnc.vnc", "enabled", "true"),
					resource.TestCheckResourceAttr("contabo_instance_vnc.vnc", "vnc_port", "63031"),
					func(s *terraform.State) error {
						if got := api.vnc[instanceId]["_password"]; got != "Secret123" {
							return fmt.Errorf("password not set: %v", got)
						}
						return nil
					},
				),
			},
			{
				Config: config("Secret456"),
				Check: func(s *terraform.State) error {
					if got := api.vnc[instanceId]["_password"]; got != "Secret456" {
						return fmt.Errorf("password not updated: %v", got)
					}
					return nil
				},
			},
		},
	})
}

// resourcesOnly removes the data blocks from a test configuration. Data
// blocks in these tests are flat and end with a closing brace in column 0.
func resourcesOnly(config string) string {
	var out []string
	skipping := false
	for _, line := range strings.Split(config, "\n") {
		if strings.HasPrefix(line, "data \"") {
			skipping = true
		}
		if !skipping {
			out = append(out, line)
		}
		if skipping && line == "}" {
			skipping = false
		}
	}
	return strings.Join(out, "\n")
}
