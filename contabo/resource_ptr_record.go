package contabo

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	uuid "github.com/satori/go.uuid"
)

const defaultPtrRecordTtl = 86400

func resourcePtrRecord() *schema.Resource {
	return &schema.Resource{
		Description: "Manage the reverse DNS (PTR) record of an IP address assigned to your account. " +
			"IPv4 addresses always have a PTR record, so creating this resource updates the existing record and destroying it leaves the record unchanged. " +
			"For IPv6 addresses the record is created and deleted.",
		CreateContext: resourcePtrRecordCreate,
		ReadContext:   resourcePtrRecordRead,
		UpdateContext: resourcePtrRecordUpdate,
		DeleteContext: resourcePtrRecordDelete,
		CustomizeDiff: resourcePtrRecordCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: resourcePtrRecordImport,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the PTR record, which is the IP address.",
			},
			"ip_address": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validation.IsIPAddress,
				DiffSuppressFunc: suppressEquivalentIp,
				Description:      "The IPv4 or IPv6 address the PTR record belongs to.",
			},
			"ptr": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressEquivalentDnsName,
				Description:      "The host name the IP address resolves to, e.g. `mail.example.com`.",
			},
			"ttl": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "The time to live of the PTR record in seconds. Can only be set for IPv6 addresses, defaults to `86400`.",
			},
		},
	}
}

func isIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() != nil
}

func resourcePtrRecordCustomizeDiff(
	ctx context.Context,
	d *schema.ResourceDiff,
	m interface{},
) error {
	if !d.NewValueKnown("ip_address") || !d.NewValueKnown("ttl") {
		return nil
	}
	_, ttlSet := d.GetOk("ttl")
	if ttlSet && d.HasChange("ttl") && isIPv4(d.Get("ip_address").(string)) {
		return fmt.Errorf("ttl can only be set for IPv6 PTR records")
	}
	return nil
}

func resourcePtrRecordCreate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	ipAddress := net.ParseIP(d.Get("ip_address").(string)).String()
	ptr := normalizeDnsName(d.Get("ptr").(string))

	// IPv4 addresses always have a PTR record which can only be updated.
	if isIPv4(ipAddress) {
		if diags := updatePtrRecord(ctx, client, ipAddress, ptr, diags); diags.HasError() {
			return diags
		}
		d.SetId(ipAddress)
		return resourcePtrRecordRead(ctx, d, m)
	}

	ttl := defaultPtrRecordTtl
	if v, ok := d.GetOk("ttl"); ok {
		ttl = v.(int)
	}

	createPtrRecordRequest := openapi.NewCreatePtrRecordRequestWithDefaults()
	createPtrRecordRequest.Ip = ipAddress
	createPtrRecordRequest.Ptr = ptr
	createPtrRecordRequest.Ttl = int64(ttl)

	_, httpResp, err := client.DNSApi.
		CreatePtrRecord(ctx).
		XRequestId(uuid.NewV4().String()).
		CreatePtrRecordRequest(*createPtrRecordRequest).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusConflict {
			return append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "PTR record already exists",
				Detail: fmt.Sprintf(
					"A PTR record for %s already exists. Import it with `terraform import <address> %s` to manage it.",
					ipAddress, ipAddress),
			})
		}
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId(ipAddress)
	return resourcePtrRecordRead(ctx, d, m)
}

func resourcePtrRecordRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	res, httpResp, err := client.DNSApi.
		RetrievePtrRecord(ctx, d.Id()).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		if isNotFound(httpResp) {
			d.SetId("")
			return diags
		}
		return HandleResponseErrors(diags, httpResp)
	}

	if len(res.Data) == 0 {
		d.SetId("")
		return diags
	} else if len(res.Data) > 1 {
		return MultipleDataObjectsError(diags)
	}

	return AddPtrRecordToData(res.Data[0], d, diags)
}

func resourcePtrRecordUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	if d.HasChange("ptr") {
		ptr := normalizeDnsName(d.Get("ptr").(string))
		if diags := updatePtrRecord(ctx, client, d.Id(), ptr, diags); diags.HasError() {
			return diags
		}
	}

	return resourcePtrRecordRead(ctx, d, m)
}

func resourcePtrRecordDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	// PTR records of IPv4 addresses cannot be deleted, so the record is only
	// removed from the state.
	if isIPv4(d.Id()) {
		d.SetId("")
		return append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "IPv4 PTR record left unchanged",
			Detail:   "PTR records of IPv4 addresses cannot be deleted. The record was removed from the Terraform state but keeps its current value.",
		})
	}

	httpResp, err := client.DNSApi.
		DeletePtrRecord(ctx, d.Id()).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil && !isNotFound(httpResp) {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")
	return diags
}

func resourcePtrRecordImport(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) ([]*schema.ResourceData, error) {
	ip := net.ParseIP(strings.TrimSpace(d.Id()))
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address %q", d.Id())
	}
	d.SetId(ip.String())
	return []*schema.ResourceData{d}, nil
}

func updatePtrRecord(
	ctx context.Context,
	client *openapi.APIClient,
	ipAddress string,
	ptr string,
	diags diag.Diagnostics,
) diag.Diagnostics {
	updatePtrRecordRequest := openapi.NewUpdatePtrRecordRequestWithDefaults()
	updatePtrRecordRequest.Ptr = ptr

	httpResp, err := client.DNSApi.
		UpdatePtrRecord(ctx, ipAddress).
		XRequestId(uuid.NewV4().String()).
		UpdatePtrRecordRequest(*updatePtrRecordRequest).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}
	return diags
}

func AddPtrRecordToData(
	ptrRecord openapi.PtrRecordResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	ipAddress := ptrRecord.Ip
	if ip := net.ParseIP(ipAddress); ip != nil {
		ipAddress = ip.String()
	}
	if err := d.Set("ip_address", ipAddress); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("ptr", ptrRecord.Ptr); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("ttl", ptrRecord.Ttl); err != nil {
		return diag.FromErr(err)
	}
	return diags
}
