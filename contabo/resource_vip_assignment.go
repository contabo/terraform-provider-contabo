package contabo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// vipResponse mirrors a virtual IP (additional or floating IP) of the API.
type vipResponse struct {
	VipId               string `json:"vipId"`
	DataCenter          string `json:"dataCenter"`
	Region              string `json:"region"`
	ResourceId          string `json:"resourceId"`
	ResourceType        string `json:"resourceType"`
	ResourceName        string `json:"resourceName"`
	ResourceDisplayName string `json:"resourceDisplayName"`
	IpVersion           string `json:"ipVersion"`
	Type                string `json:"type"`
	V4                  struct {
		Ip          string  `json:"ip"`
		NetmaskCidr float64 `json:"netmaskCidr"`
		Gateway     string  `json:"gateway"`
		Network     string  `json:"network"`
		Broadcast   string  `json:"broadcast"`
	} `json:"v4"`
}

// assigned reports whether the VIP is assigned to a resource. The API uses
// an empty string or "null" for unassigned VIPs.
func (v vipResponse) assigned() bool {
	return v.ResourceId != "" && v.ResourceId != "null" && v.ResourceType != "" && v.ResourceType != "null"
}

func retrieveVip(ctx context.Context, client *openapi.APIClient, ip string) (*vipResponse, *http.Response, error) {
	var res struct {
		Data []vipResponse `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodGet, "/v1/vips/"+url.PathEscape(ip), nil, nil, &res)
	if err != nil {
		return nil, httpResp, err
	}
	if len(res.Data) != 1 {
		return nil, httpResp, fmt.Errorf("expected one VIP with ip %s, got %d", ip, len(res.Data))
	}
	return &res.Data[0], httpResp, nil
}

func listVips(ctx context.Context, client *openapi.APIClient, query url.Values) ([]vipResponse, *http.Response, error) {
	var vips []vipResponse
	httpResp, err := listAll(ctx, client, "/v1/vips", query, func(body []byte) (int64, error) {
		var page struct {
			Data       []vipResponse `json:"data"`
			Pagination pagination    `json:"_pagination"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, err
		}
		vips = append(vips, page.Data...)
		return page.Pagination.TotalPages, nil
	})
	return vips, httpResp, err
}

func vipAssignmentPath(ip, resourceType, resourceId string) string {
	return fmt.Sprintf("/v1/vips/%s/%s/%s", url.PathEscape(ip), url.PathEscape(resourceType), url.PathEscape(resourceId))
}

func resourceVipAssignment() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource assigns a virtual IP address (VIP) to an instance or to a bare metal server. A virtual IP address is an additional or floating IPv4 address that you buy as an add-on. You order virtual IP addresses in the Customer Control Panel or with the `add_ons` setting of an instance. This resource only manages the assignment of the address. When Terraform destroys this resource, the provider removes the address from the instance.",
		CreateContext: resourceVipAssignmentCreate,
		ReadContext:   resourceVipAssignmentRead,
		DeleteContext: resourceVipAssignmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"ip": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsIPv4Address,
				Description:  "The virtual IPv4 address. This address is also the id of the resource.",
			},
			"resource_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "instances",
				ValidateFunc: validation.StringInSlice([]string{"instances", "bare-metal"}, false),
				Description:  "The type of the resource that gets the IP address. The value is `instances` or `bare-metal`. The default value is `instances`.",
			},
			"resource_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the instance or the bare metal server that gets the IP address.",
			},
			"vip_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the virtual IP address.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the virtual IP address. The value is `additional` or `floating`.",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The region of the virtual IP address.",
			},
			"data_center": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The data center of the virtual IP address.",
			},
			"netmask_cidr": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the network mask of the virtual IP address in CIDR notation.",
			},
			"gateway": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address of the gateway for the virtual IP address.",
			},
		},
	}
}

func resourceVipAssignmentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	ip := d.Get("ip").(string)
	resourceType := d.Get("resource_type").(string)
	resourceId := d.Get("resource_id").(string)

	vip, httpResp, err := retrieveVip(ctx, client, ip)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	if vip.assigned() && (vip.ResourceId != resourceId || vip.ResourceType != resourceType) {
		return diag.Errorf("VIP %s is assigned to %s %s; unassign it first", ip, vip.ResourceType, vip.ResourceId)
	}
	if !vip.assigned() {
		httpResp, err = apiRequest(ctx, client, http.MethodPost, vipAssignmentPath(ip, resourceType, resourceId), nil, nil, nil)
		if err != nil {
			return apiErrorDiags(httpResp, err)
		}
	}
	d.SetId(ip)
	return resourceVipAssignmentRead(ctx, d, m)
}

func resourceVipAssignmentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	vip, httpResp, err := retrieveVip(ctx, client, d.Id())
	if err != nil {
		if removeFromStateIfNotFound(d, httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}
	if !vip.assigned() {
		d.SetId("")
		return nil
	}
	return setSchemaValues(d, resourceVipAssignment().Schema, map[string]interface{}{
		"ip":            vip.V4.Ip,
		"resource_type": vip.ResourceType,
		"resource_id":   vip.ResourceId,
		"vip_id":        vip.VipId,
		"type":          vip.Type,
		"region":        vip.Region,
		"data_center":   vip.DataCenter,
		"netmask_cidr":  int(vip.V4.NetmaskCidr),
		"gateway":       vip.V4.Gateway,
	})
}

func resourceVipAssignmentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	path := vipAssignmentPath(d.Id(), d.Get("resource_type").(string), d.Get("resource_id").(string))
	httpResp, err := apiRequest(ctx, client, http.MethodDelete, path, nil, nil, nil)
	if err != nil && !isNotFound(httpResp) {
		return apiErrorDiags(httpResp, err)
	}
	return nil
}

func vipAttributes() map[string]*schema.Schema {
	computed := func(t schema.ValueType, description string) *schema.Schema {
		return &schema.Schema{Type: t, Computed: true, Description: description}
	}
	return map[string]*schema.Schema{
		"ip":                    computed(schema.TypeString, "The virtual IPv4 address."),
		"vip_id":                computed(schema.TypeString, "The id of the virtual IP address."),
		"type":                  computed(schema.TypeString, "The type of the virtual IP address. The value is `additional` or `floating`."),
		"ip_version":            computed(schema.TypeString, "The IP version of the address. The value is always `v4`."),
		"region":                computed(schema.TypeString, "The region of the virtual IP address."),
		"data_center":           computed(schema.TypeString, "The data center of the virtual IP address."),
		"resource_id":           computed(schema.TypeString, "The id of the resource that uses the address. The value is empty when no resource uses the address."),
		"resource_type":         computed(schema.TypeString, "The type of the resource that uses the address. The value is empty when no resource uses the address."),
		"resource_name":         computed(schema.TypeString, "The name of the resource that uses the address."),
		"resource_display_name": computed(schema.TypeString, "The display name of the resource that uses the address."),
		"netmask_cidr":          computed(schema.TypeInt, "The size of the network mask in CIDR notation."),
		"gateway":               computed(schema.TypeString, "The IP address of the gateway."),
		"network":               computed(schema.TypeString, "The network address."),
		"broadcast":             computed(schema.TypeString, "The broadcast address."),
	}
}

func flattenVip(vip vipResponse) map[string]interface{} {
	resourceId, resourceType := vip.ResourceId, vip.ResourceType
	if !vip.assigned() {
		resourceId, resourceType = "", ""
	}
	return map[string]interface{}{
		"ip":                    vip.V4.Ip,
		"vip_id":                vip.VipId,
		"type":                  vip.Type,
		"ip_version":            vip.IpVersion,
		"region":                vip.Region,
		"data_center":           vip.DataCenter,
		"resource_id":           resourceId,
		"resource_type":         resourceType,
		"resource_name":         vip.ResourceName,
		"resource_display_name": vip.ResourceDisplayName,
		"netmask_cidr":          int(vip.V4.NetmaskCidr),
		"gateway":               vip.V4.Gateway,
		"network":               vip.V4.Network,
		"broadcast":             vip.V4.Broadcast,
	}
}

func dataSourceVips() *schema.Resource {
	return &schema.Resource{
		Description: "This data source lists the virtual IP addresses of your account. Virtual IP addresses are additional or floating IPv4 addresses. You can filter the list.",
		ReadContext: dataSourceVipsRead,
		Schema: map[string]*schema.Schema{
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"additional", "floating"}, false),
				Description:  "When you set this filter, the list only contains addresses of this type. The value is `additional` or `floating`.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the list only contains addresses in this region.",
			},
			"resource_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the list only contains addresses that this resource uses.",
			},
			"unassigned": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When you set this value to `true`, the list only contains addresses that no resource uses.",
			},
			"ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The IP addresses that match the filters.",
			},
			"vips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The virtual IP addresses that match the filters, with all details.",
				Elem:        &schema.Resource{Schema: vipAttributes()},
			},
		},
	}
}

func dataSourceVipsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	query := url.Values{}
	if v := d.Get("type").(string); v != "" {
		query.Set("type", v)
	}
	if v := d.Get("resource_id").(string); v != "" {
		query.Set("resourceId", v)
	}
	vips, httpResp, err := listVips(ctx, client, query)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	region := d.Get("region").(string)
	resourceId := d.Get("resource_id").(string)
	vipType := d.Get("type").(string)
	unassigned := d.Get("unassigned").(bool)

	ips := []string{}
	out := []interface{}{}
	for _, vip := range vips {
		if region != "" && !strings.EqualFold(vip.Region, region) {
			continue
		}
		if resourceId != "" && vip.ResourceId != resourceId {
			continue
		}
		if vipType != "" && vip.Type != vipType {
			continue
		}
		if unassigned && vip.assigned() {
			continue
		}
		ips = append(ips, vip.V4.Ip)
		out = append(out, flattenVip(vip))
	}
	d.SetId(fmt.Sprintf("vips-%d", hashStrings(append([]string{vipType, region, resourceId, fmt.Sprint(unassigned)}, ips...))))
	if err := d.Set("ips", ips); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("vips", out); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func dataSourceVip() *schema.Resource {
	attributes := vipAttributes()
	attributes["ip"] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.IsIPv4Address,
		Description:  "The virtual IPv4 address that the data source reads.",
	}
	return &schema.Resource{
		Description: "This data source reads one virtual IP address. A virtual IP address is an additional or floating IPv4 address.",
		ReadContext: dataSourceVipRead,
		Schema:      attributes,
	}
}

func dataSourceVipRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	vip, httpResp, err := retrieveVip(ctx, client, d.Get("ip").(string))
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	d.SetId(vip.V4.Ip)
	return setSchemaValues(d, dataSourceVip().Schema, flattenVip(*vip))
}
