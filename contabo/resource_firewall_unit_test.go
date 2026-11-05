package contabo

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestParseFirewallInstances(t *testing.T) {
	cases := map[string][]int64{
		`"[203235786]"`:                  {203235786},
		`"[1, 2]"`:                       {1, 2},
		`"[]"`:                           {},
		`""`:                             nil,
		`null`:                           nil,
		`[3,4]`:                          {3, 4},
		`[{"instanceId":5,"name":"vm"}]`: {5},
	}
	for raw, want := range cases {
		got := parseFirewallInstances(json.RawMessage(raw))
		if len(got) == 0 && len(want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("parseFirewallInstances(%s) = %v, want %v", raw, got, want)
		}
	}
}

func TestUnitFirewallLifecycle(t *testing.T) {
	api := newFakeAPI(t)
	first := api.addInstance(nil)
	second := api.addInstance(nil)
	third := api.addInstance(nil)

	config := func(instanceIds string, extraRule string) string {
		return api.providerConfig() + fmt.Sprintf(`
resource "contabo_firewall" "test" {
  name         = "test"
  description  = "managed by terraform"
  status       = "active"
  instance_ids = [%s]

  rules {
    inbound {
      protocol     = "tcp"
      display_name = "ssh"
      dest_ports   = ["22"]
      src_cidr {
        ipv4 = ["0.0.0.0/0"]
        ipv6 = ["::/0"]
      }
    }
    inbound {
      protocol = "any"
      action   = "accept"
      status   = "active"
      src_cidr {
        ipv4 = ["10.0.0.0/8"]
      }
    }
    %s
  }
}

data "contabo_firewall" "by_name" {
  name = contabo_firewall.test.name
}
`, instanceIds, extraRule)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config(fmt.Sprintf("%d, %d", first, second), ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_firewall.test", "instance_ids.#", "2"),
					resource.TestCheckResourceAttr("contabo_firewall.test", "instances_status.#", "2"),
					resource.TestCheckResourceAttr("contabo_firewall.test", "instances_status.0.status", "ok"),
					resource.TestCheckResourceAttr("contabo_firewall.test", "rules.0.inbound.#", "2"),
					resource.TestCheckResourceAttr("contabo_firewall.test", "rules.0.inbound.0.action", "accept"),
					resource.TestCheckResourceAttr("contabo_firewall.test", "rules.0.inbound.0.status", "active"),
					resource.TestCheckResourceAttr("contabo_firewall.test", "rules.0.inbound.1.protocol", "any"),
					resource.TestCheckResourceAttr("data.contabo_firewall.by_name", "instance_ids.#", "2"),
					resource.TestCheckResourceAttr("data.contabo_firewall.by_name", "rules.0.inbound.0.display_name", "ssh"),
				),
			},
			{
				// Moving from {first, second} to {second, third} must not
				// unassign the instance that stays.
				PreConfig: func() {
					api.mu.Lock()
					api.calls = nil
					api.mu.Unlock()
				},
				Config: config(fmt.Sprintf("%d, %d", second, third), `inbound {
      protocol   = "udp"
      action     = "drop"
      dest_ports = ["53"]
      src_cidr {}
    }`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contabo_firewall.test", "instance_ids.#", "2"),
					resource.TestCheckResourceAttr("contabo_firewall.test", "rules.0.inbound.#", "3"),
					resource.TestCheckResourceAttr("contabo_firewall.test", "rules.0.inbound.2.action", "drop"),
					func(s *terraform.State) error {
						if n := api.callCount("DELETE /v1/firewalls/fw-"); n != 1 {
							return fmt.Errorf("expected one unassignment, got %d", n)
						}
						if n := api.callCount("POST /v1/firewalls/fw-"); n != 1 {
							return fmt.Errorf("expected one assignment, got %d", n)
						}
						return nil
					},
				),
			},
			{
				ResourceName:      "contabo_firewall.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Older SDK versions do not write the timeouts block on import.
				ImportStateVerifyIgnore: []string{"timeouts", "%"},
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			if len(api.firewalls) != 0 {
				return fmt.Errorf("firewall not deleted")
			}
			return nil
		},
	})
}
