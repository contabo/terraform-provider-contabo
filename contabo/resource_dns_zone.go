package contabo

import (
	"context"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func resourceDnsZone() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage a DNS zone hosted on the Contabo name servers. After the zone is created, point the domain's name servers at the ones listed in `name_servers` at your registrar.",
		CreateContext: resourceDnsZoneCreate,
		ReadContext:   resourceDnsZoneRead,
		DeleteContext: resourceDnsZoneDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceDnsZoneImport,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the DNS zone, which is the zone name.",
			},
			"zone_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressEquivalentDnsName,
				Description:      "The name of the zone, e.g. `example.com`.",
			},
			"name_servers": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The name servers authoritative for the zone. Delegate the domain to these at your registrar.",
			},
		},
	}
}

func resourceDnsZoneCreate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	zoneName := normalizeDnsName(d.Get("zone_name").(string))

	createDnsZoneRequest := openapi.NewCreateDnsZoneRequestWithDefaults()
	createDnsZoneRequest.ZoneName = zoneName

	_, httpResp, err := client.DNSApi.
		CreateDnsZone(ctx).
		XRequestId(uuid.NewV4().String()).
		CreateDnsZoneRequest(*createDnsZoneRequest).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId(zoneName)
	return resourceDnsZoneRead(ctx, d, m)
}

func resourceDnsZoneRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	zoneName := d.Id()

	httpResp, err := client.DNSApi.
		RetrieveDnsZone(ctx, zoneName).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		if isNotFound(httpResp) {
			d.SetId("")
			return diags
		}
		return HandleResponseErrors(diags, httpResp)
	}

	records, httpResp, err := listDnsZoneRecords(ctx, client, zoneName)
	if err != nil {
		if isNotFound(httpResp) {
			d.SetId("")
			return diags
		}
		if httpResp == nil {
			return diag.FromErr(err)
		}
		return HandleResponseErrors(diags, httpResp)
	}

	return AddDnsZoneToData(zoneName, records, d, diags)
}

func resourceDnsZoneImport(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) ([]*schema.ResourceData, error) {
	d.SetId(normalizeDnsName(d.Id()))
	return []*schema.ResourceData{d}, nil
}

func resourceDnsZoneDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	httpResp, err := client.DNSApi.
		DeleteDnsZone(ctx, d.Id()).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil && !isNotFound(httpResp) {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")
	return diags
}

func AddDnsZoneToData(
	zoneName string,
	records []dnsRecord,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	if err := d.Set("zone_name", zoneName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name_servers", zoneNameServers(records, zoneName)); err != nil {
		return diag.FromErr(err)
	}
	return diags
}
