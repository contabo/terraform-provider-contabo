package contabo

import (
	"context"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDnsZone() *schema.Resource {
	return &schema.Resource{
		Description: "Look up a DNS zone hosted on the Contabo name servers.",
		ReadContext: dataSourceDnsZoneRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the DNS zone, which is the zone name.",
			},
			"zone_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the zone, e.g. `example.com`.",
			},
			"name_servers": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The name servers authoritative for the zone.",
			},
		},
	}
}

func dataSourceDnsZoneRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	zoneName := normalizeDnsName(d.Get("zone_name").(string))

	records, httpResp, err := listDnsZoneRecords(ctx, client, zoneName)
	if err != nil {
		if httpResp == nil {
			return diag.FromErr(err)
		}
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId(zoneName)
	return AddDnsZoneToData(zoneName, records, d, diags)
}
