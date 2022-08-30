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
		Description:   "The Compute Management API allows you to manage compute resources (e.g. creation, deletion, starting, stopping) as well as managing snapshots and custom images. It also supports [cloud-init](https://cloud-init.io/) at least on our default images (for custom images you will need to provide cloud-init support packages). The API offers providing cloud-init scripts via the user_data field. Custom images must be provided in .qcow2 or .iso format.",
		CreateContext: resourceInstanceCreate,
		ReadContext:   resourceInstanceRead,
		UpdateContext: resourceInstanceUpdate,
		DeleteContext: resourceInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the compute instance. Use it to manage it!",
			},
			"last_updated": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Time of the last update of the compute instance.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the compute instance.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Image Id is used to set up the compute instance. Ubuntu 20.04 is the default.",
			},
			"image_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Image Id is used to set up the compute instance. Ubuntu 20.04 is the default.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Instance Region where the compute instance should be located. Default region is the EU. Following regions are available: `EU`,`US-central`,`US-east`,`US-west`,`SIN`.",
			},
			"product_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Choose the VPS/VDS product you want to buy. See our products [here](https://api.contabo.com/#tag/Instances/operation/createInstance).",
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
										Type:        schema.TypeString,
										Computed:    true,
										Description: "IP Address",
									},
									"netmask_cidr": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Netmask CIDR",
									},
									"gateway": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Gateway",
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
										Type:        schema.TypeString,
										Computed:    true,
										Description: "IP Address",
									},
									"netmask_cidr": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Netmask CIDR",
									},
									"gateway": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Gateway",
									},
								},
							},
						},
					},
				},
			},
			"mac_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Mac address of the instance.",
			},
			"ram_mb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Image ram size in megabyte.",
			},
			"cpu_cores": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "CPU core count of the instance.",
			},
			"disk_mb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Image disk size of the instance in megabyte.",
			},
			"os_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Type of operating system (OS) installed on the instance.",
			},
			"ssh_keys": {
				Computed: true,
				Optional: true,
				Type:     schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "Array of `secretIds` of public SSH keys for logging into as defaultUser with administrator/root privileges. Applies to Linux/BSD systems. Please refer to Secrets Management API.",
			},
			"root_password": {
				Optional:    true,
				Type:        schema.TypeInt,
				Description: "Root password of the compute instance.",
			},
			"created_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The creation date of the compute instance.",
			},
			"cancel_date": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The date on which the instance will be cancelled.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the compute instance. The status can be set to `provisioning`, `uninstalled`, `running`, `stopped`, `error`, `installing`, `unknown`, or `installed`.",
			},
			"v_host_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Identifier of the host system.",
			},
			"add_ons": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional:    true,
							Description: "Id of the Addon. Please refer to list [here](https://contabo.com/en/product-list/?show_ids=true).",
						},
						"quantity": {
							Type:        schema.TypeInt,
							Computed:    true,
							Optional:    true,
							Description: "The number of Addons you wish to aquire.",
						},
					},
				},
			},
			"error_message": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "If the instance is in an error state (see status property), the error message can be seen in this field.",
			},
			"product_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "InsInstance's category depending on Product Id. Following product types are available: `hdd`,`ssd`,`vds`,`nvme`.",
			},
			"user_data": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Cloud-Init Config in order to customize during start of compute instance.",
			},
			"license": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Additional license in order to enhance your chosen product. It is mainly needed for software licenses on your product (not needed for windows). See our [api documentation](https://api.contabo.com/#tag/Instances/operation/createInstance) for all available licenses.",
			},
			"period": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Initial contract period in months. Available periods are: 1, 3, 6 and 12 months. The default setting is 1 month.",
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
		createInstanceRequest.ProductId = productId
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
