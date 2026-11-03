package contabo

import (
	"context"
	"fmt"
	"net"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	uuid "github.com/satori/go.uuid"
)

func dataSourcePtrRecord() *schema.Resource {
	return &schema.Resource{
		Description: "Look up the reverse DNS (PTR) record of an IP address assigned to your account.",
		ReadContext: dataSourcePtrRecordRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the PTR record, which is the IP address.",
			},
			"ip_address": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsIPAddress,
				Description:  "The IPv4 or IPv6 address to look up.",
			},
			"ptr": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The host name the IP address resolves to.",
			},
			"ttl": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The time to live of the PTR record in seconds.",
			},
		},
	}
}

func dataSourcePtrRecordRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	ipAddress := net.ParseIP(d.Get("ip_address").(string)).String()

	res, httpResp, err := client.DNSApi.
		RetrievePtrRecord(ctx, ipAddress).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	if len(res.Data) == 0 {
		return HandleMissingDataObjectsFilters(diags, "PTR record not found",
			fmt.Sprintf("No PTR record found for %s.", ipAddress))
	} else if len(res.Data) > 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(ipAddress)
	if err := d.Set("ptr", res.Data[0].Ptr); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("ttl", res.Data[0].Ttl); err != nil {
		return diag.FromErr(err)
	}
	return diags
}
