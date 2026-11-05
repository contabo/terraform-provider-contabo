package contabo

import (
	"context"
	"net/url"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFirewall() *schema.Resource {
	return &schema.Resource{
		Description: "This data source reads one firewall by the id or by the name of the firewall.",
		ReadContext: dataSourceFirewallRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"id", "name"},
				Description:  "The id of the firewall. Set either `id` or `name`.",
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"id", "name"},
				Description:  "The name of the firewall. Set either `id` or `name`. Exactly one firewall must have this name.",
			},
			"created_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date when the firewall was created.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the firewall.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the firewall. The value is `active` or `inactive`.",
			},
			"instance_ids": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Computed:    true,
				Description: "The ids of the instances that the firewall protects.",
			},
			"instances_status": firewallInstancesStatusSchema(),
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The rules of the firewall.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"inbound": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The inbound rules of the firewall.",
							Elem:        firewallRuleSchema(true),
						},
					},
				},
			},
		},
	}
}

func dataSourceFirewallRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)

	if firewallId := d.Get("id").(string); firewallId != "" {
		firewall, httpResp, err := retrieveFirewall(ctx, client, firewallId)
		if err != nil {
			return apiErrorDiags(httpResp, err)
		}
		return setFirewallData(*firewall, d)
	}

	name := d.Get("name").(string)
	firewalls, httpResp, err := listFirewalls(ctx, client, url.Values{"name": []string{name}})
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	var matches []firewallResponse
	for _, firewall := range firewalls {
		if firewall.Name == name {
			matches = append(matches, firewall)
		}
	}
	switch len(matches) {
	case 0:
		return diag.Errorf("no firewall named %q", name)
	case 1:
		return setFirewallData(matches[0], d)
	default:
		return diag.Errorf("%d firewalls are named %q; look the firewall up by id", len(matches), name)
	}
}
