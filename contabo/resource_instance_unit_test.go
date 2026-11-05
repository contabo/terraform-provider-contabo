package contabo

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func instanceIdFromState(s *terraform.State, name string) (int64, error) {
	rs, ok := s.RootModule().Resources[name]
	if !ok {
		return 0, fmt.Errorf("%s not in state", name)
	}
	return strconv.ParseInt(rs.Primary.ID, 10, 64)
}

func TestUnitInstanceLifecycle(t *testing.T) {
	api := newFakeAPI(t)
	var instanceId int64

	config := func(imageId string, userData string, displayName string, addOns string) string {
		return api.providerConfig() + fmt.Sprintf(`
resource "contabo_instance" "test" {
  display_name = %q
  product_id   = "V153"
  region       = "EU"
  image_id     = %q
  user_data    = %q
  default_user = "root"
  ssh_keys     = [11, 12]
  %s
}
`, displayName, imageId, userData, addOns)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("image-a", "#cloud-config\n", "node-1", `add_ons {
    id = "1477"
  }`),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						var err error
						instanceId, err = instanceIdFromState(s, "contabo_instance.test")
						return err
					},
					resource.TestCheckResourceAttr("contabo_instance.test", "status", "running"),
					resource.TestCheckResourceAttr("contabo_instance.test", "default_user", "root"),
					resource.TestCheckResourceAttr("contabo_instance.test", "ssh_keys.#", "2"),
					resource.TestCheckResourceAttr("contabo_instance.test", "add_ons.#", "1"),
					resource.TestCheckResourceAttr("contabo_instance.test", "add_ons.0.id", "1477"),
					resource.TestCheckResourceAttr("contabo_instance.test", "product_name", "Cloud VPS 4"),
					resource.TestCheckResourceAttr("contabo_instance.test", "ip_config.0.v4.0.ip", "203.0.113.10"),
					func(s *terraform.State) error {
						if len(api.creates) != 1 {
							return fmt.Errorf("expected one create request, got %d", len(api.creates))
						}
						addOns, _ := api.creates[0]["addOns"].(map[string]interface{})
						if addOns == nil || addOns["addonsIds"] == nil {
							return fmt.Errorf("add-ons not sent on create: %v", api.creates[0])
						}
						if api.creates[0]["period"] != float64(1) {
							return fmt.Errorf("expected period 1, got %v", api.creates[0]["period"])
						}
						return nil
					},
				),
			},
			{
				// Changing only the image reinstalls with the full set of
				// reinstall values (issues #45 and #49).
				Config: config("image-b", "#cloud-config\n", "node-1", `add_ons {
    id = "1477"
  }`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_instance.test", "image_id", "image-b"),
					resource.TestCheckResourceAttr("contabo_instance.test", "default_user", "root"),
					func(s *terraform.State) error {
						if len(api.reinstalls) != 1 {
							return fmt.Errorf("expected one reinstall, got %d", len(api.reinstalls))
						}
						req := api.reinstalls[0]
						if req["userData"] != "#cloud-config\n" {
							return fmt.Errorf("user_data not sent on reinstall: %v", req)
						}
						if req["defaultUser"] != "root" {
							return fmt.Errorf("default_user not sent on reinstall: %v", req)
						}
						if keys, _ := req["sshKeys"].([]interface{}); len(keys) != 2 {
							return fmt.Errorf("ssh_keys not sent on reinstall: %v", req)
						}
						return nil
					},
				),
			},
			{
				// A display name change patches without reinstalling, and
				// adding private networking and backup uses the upgrade API.
				Config: config("image-b", "#cloud-config\n", "node-renamed", `add_ons {
    id = "1477"
  }
  add_ons {
    id = "2214"
  }`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_instance.test", "display_name", "node-renamed"),
					resource.TestCheckResourceAttr("contabo_instance.test", "add_ons.#", "2"),
					func(s *terraform.State) error {
						if len(api.reinstalls) != 1 {
							return fmt.Errorf("display name change reinstalled the instance")
						}
						if len(api.upgrades) != 1 {
							return fmt.Errorf("expected one upgrade, got %d", len(api.upgrades))
						}
						if _, ok := api.upgrades[0]["backup"]; !ok {
							return fmt.Errorf("backup not requested: %v", api.upgrades[0])
						}
						if _, ok := api.upgrades[0]["privateNetworking"]; ok {
							return fmt.Errorf("private networking requested although present: %v", api.upgrades[0])
						}
						return nil
					},
				),
			},
			{
				// Add-ons the upgrade API cannot order fail at plan time.
				Config: config("image-b", "#cloud-config\n", "node-renamed", `add_ons {
    id = "1477"
  }
  add_ons {
    id = "9999"
  }`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`cannot be added to existing instance`),
			},
			{
				// product_id changes fail at plan time instead of being ignored.
				Config:      replaceOnce(config("image-b", "#cloud-config\n", "node-renamed", ""), `"V153"`, `"V154"`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`product_id cannot be changed`),
			},
			{
				ResourceName:            "contabo_instance.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"user_data", "add_ons", "last_updated", "cancel_on_destroy", "period", "license", "existing_instance_id"},
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			if api.instance(instanceId)["cancelDate"] == nil {
				return fmt.Errorf("instance %d was not cancelled", instanceId)
			}
			return nil
		},
	})
}

func TestUnitInstanceAdoptAndKeepOnDestroy(t *testing.T) {
	api := newFakeAPI(t)
	existing := api.addInstance(map[string]interface{}{"displayName": "old", "defaultUser": "root"})
	api.notCancelable[existing] = true

	config := func(cancel bool) string {
		return api.providerConfig() + fmt.Sprintf(`
resource "contabo_instance" "adopted" {
  existing_instance_id = "%d"
  display_name         = "adopted"
  image_id             = "some-other-image"
  cancel_on_destroy    = %t
}
`, existing, cancel)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config(false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_instance.adopted", "id", strconv.FormatInt(existing, 10)),
					resource.TestCheckResourceAttr("contabo_instance.adopted", "display_name", "adopted"),
					func(s *terraform.State) error {
						if len(api.reinstalls) != 0 {
							return fmt.Errorf("adoption reinstalled the instance")
						}
						return nil
					},
				),
				// image_id differs from the adopted instance, so the next
				// plan reinstalls; that is expected and documented.
				ExpectNonEmptyPlan: true,
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			if api.instance(existing)["cancelDate"] != nil {
				return fmt.Errorf("instance was cancelled although cancel_on_destroy is false")
			}
			if api.callCount("POST /v1/compute/instances/") != 0 {
				return fmt.Errorf("cancel was called")
			}
			return nil
		},
	})
}

func TestUnitInstanceCancelErrorHint(t *testing.T) {
	api := newFakeAPI(t)
	existing := api.addInstance(nil)
	api.notCancelable[existing] = true

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: api.providerConfig() + fmt.Sprintf(`
resource "contabo_instance" "adopted" {
  existing_instance_id = "%d"
}
`, existing),
			},
			{
				Config:      api.providerConfig(),
				ExpectError: regexp.MustCompile(`cancel_on_destroy = false`),
			},
			{
				// Let the test framework clean up.
				PreConfig: func() { api.notCancelable[existing] = false },
				Config:    api.providerConfig(),
			},
		},
	})
}

func TestUnitInstanceRemovedOutsideTerraform(t *testing.T) {
	api := newFakeAPI(t)
	var instanceId int64
	config := api.providerConfig() + `
resource "contabo_instance" "test" {
  product_id = "V153"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					var err error
					instanceId, err = instanceIdFromState(s, "contabo_instance.test")
					return err
				},
			},
			{
				PreConfig: func() {
					api.mu.Lock()
					delete(api.instances, instanceId)
					api.mu.Unlock()
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitInstancesDataSource(t *testing.T) {
	api := newFakeAPI(t)
	first := api.addInstance(map[string]interface{}{"displayName": "k8s-cp-1", "productId": "V159"})
	api.addInstance(map[string]interface{}{"displayName": "k8s-worker-1", "productId": "V160"})
	api.addInstance(map[string]interface{}{"displayName": "other", "productId": "V160", "status": "stopped"})
	api.tagAssignments[7] = []map[string]interface{}{
		{"tagId": 7, "resourceType": "instance", "resourceId": strconv.FormatInt(first, 10)},
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: api.providerConfig() + `
data "contabo_instances" "k8s" {
  display_name_prefix = "k8s-"
}
data "contabo_instances" "running_v160" {
  product_ids = ["V160"]
  statuses    = ["running"]
}
data "contabo_instances" "tagged" {
  tag_ids = [7]
}
data "contabo_instance" "one" {
  id = data.contabo_instances.tagged.ids[0]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.contabo_instances.k8s", "instances.#", "2"),
					resource.TestCheckResourceAttr("data.contabo_instances.k8s", "instances.0.display_name", "k8s-cp-1"),
					resource.TestCheckResourceAttr("data.contabo_instances.running_v160", "instances.#", "1"),
					resource.TestCheckResourceAttr("data.contabo_instances.running_v160", "instances.0.display_name", "k8s-worker-1"),
					resource.TestCheckResourceAttr("data.contabo_instances.tagged", "ids.#", "1"),
					resource.TestCheckResourceAttr("data.contabo_instances.tagged", "ids.0", strconv.FormatInt(first, 10)),
					resource.TestCheckResourceAttr("data.contabo_instance.one", "product_id", "V159"),
					resource.TestCheckResourceAttr("data.contabo_instance.one", "add_ons.0.id", "1501"),
				),
			},
		},
	})
}

func replaceOnce(s, old, new string) string {
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func TestInstanceStateUpgradeV0(t *testing.T) {
	state, err := resourceInstanceStateUpgradeV0(context.Background(), map[string]interface{}{"id": "1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if state["cancel_on_destroy"] != true {
		t.Fatalf("cancel_on_destroy not defaulted: %v", state)
	}
	state, _ = resourceInstanceStateUpgradeV0(context.Background(), map[string]interface{}{"cancel_on_destroy": false}, nil)
	if state["cancel_on_destroy"] != false {
		t.Fatalf("existing value overwritten: %v", state)
	}
	if !resourceInstanceV0Type().IsObjectType() {
		t.Fatal("v0 type is not an object")
	}
}
