package contabo

import (
	"context"
	"fmt"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	uuid "github.com/satori/go.uuid"
)

func resourceDnsRecord() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage a record in a DNS zone hosted on the Contabo name servers. Supported record types are A, AAAA, CAA, CNAME, MX, SRV and TXT. The NS and SOA records of a zone are managed by Contabo.",
		CreateContext: resourceDnsRecordCreate,
		ReadContext:   resourceDnsRecordRead,
		UpdateContext: resourceDnsRecordUpdate,
		DeleteContext: resourceDnsRecordDelete,
		CustomizeDiff: resourceDnsRecordCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the record in the form `<zone_name>/<record_id>`. Use it to import an existing record.",
			},
			"zone_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressEquivalentDnsName,
				Description:      "The name of the zone the record belongs to.",
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					zoneName := d.Get("zone_name").(string)
					return relativeDnsRecordName(old, zoneName) == relativeDnsRecordName(new, zoneName)
				},
				Description: "The record name relative to the zone, e.g. `www`. Leave empty or use `@` for the zone apex. A fully qualified name inside the zone is accepted as well.",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The fully qualified name of the record.",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(dnsRecordTypes, false),
				Description:  "The record type. One of `A`, `AAAA`, `CAA`, `CNAME`, `MX`, `SRV`, `TXT`.",
			},
			"data": {
				Type:     schema.TypeString,
				Required: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return dnsRecordDataEqual(d.Get("type").(string), old, new)
				},
				Description: "The record value, e.g. an IP address for `A`/`AAAA` records, a host name for `CNAME`, `MX` and `SRV` records, the text for `TXT` records or the CAA value.",
			},
			"ttl": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      3600,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "The time to live of the record in seconds.",
			},
			"priority": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The priority of the record. Only applies to `MX` and `SRV` records.",
			},
			"weight": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The weight of the record. Only applies to `SRV` records.",
			},
			"port": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				ValidateFunc: validation.IntBetween(0, 65535),
				Description:  "The port of the service. Only applies to `SRV` records.",
			},
			"flag": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				ValidateFunc: validation.IntBetween(0, 255),
				Description:  "The flag of the record. Only applies to `CAA` records.",
			},
			"tag": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.StringInSlice([]string{"", "issue", "issuewild", "iodef"}, false),
				Description:  "The property tag of the record, one of `issue`, `issuewild` or `iodef`. Required for `CAA` records and only applies to them.",
			},
		},
	}
}

func resourceDnsRecordCustomizeDiff(
	ctx context.Context,
	d *schema.ResourceDiff,
	m interface{},
) error {
	recordType := d.Get("type").(string)

	if recordType != "MX" && recordType != "SRV" && d.Get("priority").(int) != 0 {
		return fmt.Errorf("priority can only be set for MX and SRV records")
	}
	if recordType != "SRV" && (d.Get("weight").(int) != 0 || d.Get("port").(int) != 0) {
		return fmt.Errorf("weight and port can only be set for SRV records")
	}
	if recordType != "CAA" && (d.Get("flag").(int) != 0 || d.Get("tag").(string) != "") {
		return fmt.Errorf("flag and tag can only be set for CAA records")
	}
	if recordType == "CAA" && d.Get("tag").(string) == "" {
		return fmt.Errorf("tag is required for CAA records")
	}
	return nil
}

func resourceDnsRecordCreate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	zoneName := normalizeDnsName(d.Get("zone_name").(string))
	recordType := d.Get("type").(string)
	name := relativeDnsRecordName(d.Get("name").(string), zoneName)

	createDnsZoneRecordRequest := openapi.NewCreateDnsZoneRecordRequestWithDefaults()
	createDnsZoneRecordRequest.Name = &name
	createDnsZoneRecordRequest.Type = recordType
	createDnsZoneRecordRequest.Ttl = float32(d.Get("ttl").(int))
	createDnsZoneRecordRequest.Prio = float32(d.Get("priority").(int))
	createDnsZoneRecordRequest.Data = d.Get("data").(string)
	createDnsZoneRecordRequest.Port, createDnsZoneRecordRequest.Weight,
		createDnsZoneRecordRequest.Flag, createDnsZoneRecordRequest.Tag = dnsRecordTypeSpecificFields(recordType, d)

	_, httpResp, err := client.DNSApi.
		CreateDnsZoneRecord(ctx, zoneName).
		XRequestId(uuid.NewV4().String()).
		CreateDnsZoneRecordRequest(*createDnsZoneRecordRequest).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	res, err := decodeDnsRecordsResponse(httpResp)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(res.Data) == 0 {
		return NoDataError(diags)
	} else if len(res.Data) > 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(dnsRecordId(zoneName, res.Data[0].RecordId))
	return resourceDnsRecordRead(ctx, d, m)
}

func resourceDnsRecordRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	zoneName, recordId, err := parseDnsRecordId(d.Id())
	if err != nil {
		return diag.FromErr(err)
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

	for _, record := range records {
		if record.RecordId == recordId {
			return AddDnsRecordToData(zoneName, record, d, diags)
		}
	}

	d.SetId("")
	return diags
}

func resourceDnsRecordUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	zoneName, recordId, err := parseDnsRecordId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.HasChanges("data", "ttl", "priority", "weight", "port", "flag", "tag") {
		return diags
	}

	// The API replaces all mutable attributes, so always send the full set.
	recordType := d.Get("type").(string)
	updateDnsZoneRecordRequest := openapi.NewUpdateDnsZoneRecordRequestWithDefaults()
	updateDnsZoneRecordRequest.Type = recordType
	updateDnsZoneRecordRequest.Ttl = float32(d.Get("ttl").(int))
	updateDnsZoneRecordRequest.Prio = float32(d.Get("priority").(int))
	updateDnsZoneRecordRequest.Data = d.Get("data").(string)
	updateDnsZoneRecordRequest.Port, updateDnsZoneRecordRequest.Weight,
		updateDnsZoneRecordRequest.Flag, updateDnsZoneRecordRequest.Tag = dnsRecordTypeSpecificFields(recordType, d)

	_, httpResp, err := client.DNSApi.
		UpdateDnsZoneRecord(ctx, recordId, zoneName).
		XRequestId(uuid.NewV4().String()).
		UpdateDnsZoneRecordRequest(*updateDnsZoneRecordRequest).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	return resourceDnsRecordRead(ctx, d, m)
}

func resourceDnsRecordDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	zoneName, recordId, err := parseDnsRecordId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	httpResp, err := client.DNSApi.
		DeleteDnsZoneRecord(ctx, recordId, zoneName).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil && !isNotFound(httpResp) {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")
	return diags
}

// dnsRecordTypeSpecificFields returns port, weight, flag and tag for the
// record types they apply to, and nil otherwise.
func dnsRecordTypeSpecificFields(
	recordType string,
	d *schema.ResourceData,
) (*float32, *float32, *float32, *string) {
	switch recordType {
	case "SRV":
		port := float32(d.Get("port").(int))
		weight := float32(d.Get("weight").(int))
		return &port, &weight, nil, nil
	case "CAA":
		flag := float32(d.Get("flag").(int))
		tag := d.Get("tag").(string)
		return nil, nil, &flag, &tag
	}
	return nil, nil, nil, nil
}

func AddDnsRecordToData(
	zoneName string,
	record dnsRecord,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	values := map[string]interface{}{
		"zone_name": zoneName,
		"name":      relativeDnsRecordName(record.Name, zoneName),
		"fqdn":      fqdnDnsRecordName(record.Name, zoneName),
		"type":      record.Type,
		"data":      record.Data,
		"ttl":       floatToInt(record.Ttl),
		"priority":  floatToInt(record.Prio),
		"weight":    floatToInt(record.Weight),
		"port":      floatToInt(record.Port),
		"flag":      floatToInt(record.Flag),
		"tag":       stringValue(record.Tag),
	}
	for key, value := range values {
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	}
	return diags
}
