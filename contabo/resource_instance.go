package contabo

import (
	"context"
	"strconv"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceInstanceCreate,
		ReadContext:   resourceInstanceRead,
		UpdateContext: resourceInstanceUpdate,
		DeleteContext: resourceInstanceDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_updated": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"display_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"image_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"product_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"ip_config": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"v4": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ip": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"netmask_cidr": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"gateway": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"v6": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ip": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"netmask_cidr": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"gateway": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"mac_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ram_mb": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"cpu_cores": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"disk_mb": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"os_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ssh_keys": {
				Computed: true,
				Optional: true,
				Type:     schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"root_password": {
				Optional: true,
				Type:     schema.TypeInt,
			},
			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cancel_date": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"v_host_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"add_ons": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
						"quantity": {
							Type:     schema.TypeInt,
							Computed: true,
							Optional: true,
						},
					},
				},
			},
			"error_message": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"product_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"license": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"period": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceInstanceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	createInstanceRequest := openapi.NewCreateInstanceRequestWithDefaults()

	displayName := d.Get("display_name").(string)
	imageId := d.Get("image_id").(string)
	region := d.Get("region").(string)
	productId := d.Get("product_id").(string)
	sshKeys := d.Get("ssh_keys")
	rootPassword := d.Get("root_password")
	userData := d.Get("user_data").(string)
	license := d.Get("license").(string)
	period := d.Get("period").(int)

	if displayName != "" {
		createInstanceRequest.DisplayName = &displayName
	}
	if imageId != "" {
		createInstanceRequest.ImageId = imageId
	}
	if region != "" {
		createInstanceRequest.Region = region
	}
	if productId != "" {
		createInstanceRequest.ProductId = imageId
	}
	if sshKeys != nil {
		var sshKeys64 []int64
		for _, key := range sshKeys.([]interface{}) {
			sshKey := key.(int)
			sshKeys64 = append(sshKeys64, int64(sshKey))
		}
		createInstanceRequest.SshKeys = &sshKeys64
	}
	if rootPassword != nil {
		rootPassword64 := int64(rootPassword.(int))
		createInstanceRequest.RootPassword = &rootPassword64
	}
	if userData != "" {
		createInstanceRequest.UserData = &userData
	}
	if license != "" {
		createInstanceRequest.License = &license
	}
	if period != 0 {
		createInstanceRequest.Period = int64(period)
	}

	res, httpResp, err := client.InstancesApi.
		CreateInstance(ctx).
		XRequestId(uuid.NewV4().String()).
		CreateInstanceRequest(*createInstanceRequest).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(strconv.Itoa(int(res.Data[0].InstanceId)))

	return resourceInstanceRead(ctx, d, m)
}

func resourceInstanceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.InstancesApi.
		RetrieveInstance(ctx, instanceId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	return AddInstanceToData(res.Data[0], d, diags)
}

func resourceInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	anyChange := false
	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	patchInstanceRequest := openapi.NewReinstallInstanceRequestWithDefaults()

	if d.HasChange("ssh_keys") {
		sshKeys := d.Get("ssh_keys")
		if sshKeys != nil {
			var sshKeys64 []int64
			for _, key := range sshKeys.([]interface{}) {
				sshKey := key.(int)
				sshKeys64 = append(sshKeys64, int64(sshKey))
			}
			patchInstanceRequest.SshKeys = &sshKeys64
			anyChange = true
		}
	}

	if d.HasChange("root_password") {
		rootPassword := d.Get("root_password")
		if rootPassword != nil {
			rootPassword64 := int64(rootPassword.(int))
			patchInstanceRequest.RootPassword = &rootPassword64
			anyChange = true
		}
	}

	if d.HasChange("user_data") {
		userData := d.Get("user_data").(string)
		if userData != "" {
			patchInstanceRequest.UserData = &userData
			anyChange = true
		}
	}

	if d.HasChange("image_id") {
		imageId := d.Get("image_id").(string)
		if imageId != "" {
			patchInstanceRequest.ImageId = imageId
			anyChange = true
		}
	}

	if anyChange {
		res, httpResp, err := client.InstancesApi.
			ReinstallInstance(ctx, instanceId).
			XRequestId(uuid.NewV4().String()).
			ReinstallInstanceRequest(*patchInstanceRequest).
			Execute()

		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		} else if len(res.Data) != 1 {
			return MultipleDataObjectsError(diags)
		}

		d.SetId(strconv.Itoa(int(res.Data[0].InstanceId)))

		return resourceInstanceRead(ctx, d, m)
	}

	return diags
}

func resourceInstanceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	return diags
}

func AddInstanceToData(
	instance openapi.InstanceResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	id := strconv.Itoa(int(instance.InstanceId))
	if err := d.Set("id", id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", instance.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("display_name", instance.DisplayName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("image_id", instance.ImageId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("product_id", instance.ProductId); err != nil {
		return diag.FromErr(err)
	}
	ipConfig := BuildIpConfig(instance.IpConfig)
	if err := d.Set("ip_config", ipConfig); err != nil && len(ipConfig) > 0 {
		return diag.FromErr(err)
	}
	if err := d.Set("mac_address", instance.MacAddress); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("ram_mb", instance.RamMb); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cpu_cores", instance.CpuCores); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("disk_mb", instance.DiskMb); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("os_type", instance.OsType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("ssh_keys", instance.SshKeys); err != nil {
		return diag.FromErr(err)
	}
	createdDate := instance.CreatedDate.Format(time.RFC850)
	if err := d.Set("created_date", createdDate); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cancel_date", instance.CancelDate); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", instance.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("v_host_id", instance.VHostId); err != nil {
		return diag.FromErr(err)
	}
	addOns := BuildAddons(instance.AddOns)
	if err := d.Set("add_ons", addOns); err != nil && len(addOns) > 0 {
		return diag.FromErr(err)
	}
	if err := d.Set("error_message", instance.ErrorMessage); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("product_type", instance.ProductType); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func BuildIpConfig(ipConfigResponse *openapi.IpConfig2) []interface{} {
	if ipConfigResponse != nil {
		ipConfig := make(map[string]interface{})

		v4 := make(map[string]interface{})
		v4["ip"] = ipConfigResponse.V4.Ip
		v4["netmask_cidr"] = int(ipConfigResponse.V4.NetmaskCidr)
		v4["gateway"] = ipConfigResponse.V4.Gateway

		v6 := make(map[string]interface{})
		v6["ip"] = ipConfigResponse.V6.Ip
		v6["netmask_cidr"] = int(ipConfigResponse.V6.NetmaskCidr)
		v6["gateway"] = ipConfigResponse.V6.Gateway

		ipConfig["v4"] = []interface{}{v4}
		ipConfig["v6"] = []interface{}{v6}

		return []interface{}{ipConfig}
	}

	return nil
}

func BuildAddons(addOnResponse []openapi.AddOnResponse) []map[string]interface{} {
	if addOnResponse != nil {
		var addOns []map[string]interface{}

		for _, addOn := range addOnResponse {
			builtAddOn := make(map[string]interface{})
			builtAddOn["id"] = addOn.Id
			builtAddOn["quantity"] = addOn.Quantity

			addOns = append(addOns, builtAddOn)
		}

		return addOns
	}

	return nil
}
