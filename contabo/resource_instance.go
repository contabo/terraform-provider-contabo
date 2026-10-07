package contabo

import (
	"context"
	"strconv"
	"strings"
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
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the compute instance. Use it to manage it!",
			},
			"existing_instance_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The identifier of the existing compute instance. (override id)",
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
				Description: "The instance name chosen by the customer that will be shown in the customer panel.",
			},
			"image_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "CAUTION: On updating this value your server will be reinstalled! Image Id is used to set up the compute instance. Ubuntu 20.04 is the default, currently you have to get the Id with our [API](https://api.contabo.com/#tag/Images/operation/retrieveImage) or via our [command line](https://github.com/contabo/cntb) tool with this command: `cntb get images`.",
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
				Description: "CAUTION: On updating this value your server will be reinstalled! Array of `secretIds` of public SSH keys for logging into as defaultUser with administrator/root privileges. Applies to Linux/BSD systems. Please refer to Secrets Management API.",
			},
			"root_password": {
				Optional:    true,
				Type:        schema.TypeInt,
				Description: "CAUTION: On updating this value your server will be reinstalled! Root password of the compute instance.",
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
				Description: "CAUTION: On updating this value your server will be reinstalled! Cloud-Init Config in order to customize during start of compute instance.",
			},
			"private_network_ids": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Optional:    true,
				Description: "Set of private network IDs to attach to this instance. The instance will be automatically provisioned with the necessary virtual network interfaces.",
			},
			"license": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Additional license in order to enhance your chosen product. It is mainly needed for software licenses on your product (not needed for windows). See our [api documentation](https://api.contabo.com/#tag/Instances/operation/createInstance) for all available licenses.",
			},
			"default_user": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Default user name created for login during (re-)installation with administrative privileges. Allowed values for Linux/BSD are admin (use sudo to apply administrative privileges like root) or root. Allowed values for Windows are admin (has administrative privileges like administrator) or administrator.",
			},
			"period": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Initial contract period in months. Available periods are: 1, 3, 6 and 12 months. The default setting is 1 month.",
			},
			"additional_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "All other additional IP addresses of the instance.",
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
					},
				},
			},
		},
	}
}

func resourceInstanceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	extstingId := d.Get("existing_instance_id").(string)

	if extstingId != "" {
		d.SetId(extstingId)

		return resourceInstanceUpdate(ctx, d, m)
	}

	createInstanceRequest := openapi.NewCreateInstanceRequestWithDefaults()

	displayName := d.Get("display_name").(string)
	imageId := d.Get("image_id").(string)
	region := d.Get("region").(string)
	productId := d.Get("product_id").(string)
	sshKeys := d.Get("ssh_keys")
	rootPassword := d.Get("root_password")
	userData := d.Get("user_data").(string)
	license := d.Get("license").(string)
	defaultUser := d.Get("default_user").(string)
	period := d.Get("period").(int)

	if displayName != "" {
		createInstanceRequest.DisplayName = &displayName
	}
	if imageId != "" {
		createInstanceRequest.ImageId = &imageId
	}
	if region != "" {
		createInstanceRequest.Region = &region
	}
	if productId != "" {
		createInstanceRequest.ProductId = &productId
	}
	var sshKeys64 []int64
	if sshKeys != nil {
		for _, key := range sshKeys.([]interface{}) {
			sshKey := key.(int)
			sshKeys64 = append(sshKeys64, int64(sshKey))
		}
		createInstanceRequest.SshKeys = &sshKeys64
	}
	var rootPassword64 int64
	if rootPassword != nil {
		rootPassword64 = int64(rootPassword.(int))
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

	if defaultUser != "" {
		createInstanceRequest.DefaultUser = &defaultUser
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

	instanceId := res.Data[0].InstanceId
	d.SetId(strconv.Itoa(int(instanceId)))

	if pnIds, ok := d.GetOk("private_network_ids"); ok {
		pnSet := pnIds.(*schema.Set).List()
		var ids []int64
		for _, id := range pnSet {
			ids = append(ids, int64(id.(int)))
		}
		if len(ids) > 0 {
			pnDiag := syncPrivateNetworksStatus(ctx, diags, client, d, ids, instanceId)
			if len(pnDiag) > 0 {
				return pnDiag
			}
		}
	}

	return resourceInstanceRead(ctx, d, m)
}

func resourceInstanceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	instance, diag := pollInstanceInstalled(diags, client, ctx, instanceId)

	if err != nil || instance == nil {
		return append(diags, diag...)
	}

	return AddInstanceToData(*instance, d, diags)
}

func resourceInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("private_network_ids") {
		oldVal, newVal := d.GetChange("private_network_ids")
		oldSet := oldVal.(*schema.Set)
		newSet := newVal.(*schema.Set)

		// Unassign from removed private networks
		for _, oldId := range oldSet.List() {
			if !newSet.Contains(oldId) {
				_, _ = unassignInstanceToPrivateNetwork(diags, client, int64(oldId.(int)), instanceId)
			}
		}

		// Sync desired private networks
		var ids []int64
		for _, id := range newSet.List() {
			ids = append(ids, int64(id.(int)))
		}
		if len(ids) > 0 {
			pnDiag := syncPrivateNetworksStatus(ctx, diags, client, d, ids, instanceId)
			if len(pnDiag) > 0 {
				return pnDiag
			}
		}
	}

	if shouldUpdateInstanceValues(d) {
		updateInstanceValues(d, client, ctx, instanceId, diags, m)
	}
	if shouldReinstall(d) {
		reinstall(d, client, ctx, instanceId, diags, m)
	}
	return resourceInstanceRead(ctx, d, m)
}

func shouldUpdateInstanceValues(d *schema.ResourceData) bool {
	updateChange := false
	if d.HasChange("display_name") {
		displayname := d.Get("display_name").(string)
		if displayname != "" {
			updateChange = true
		}
	}
	return updateChange
}

func shouldReinstall(d *schema.ResourceData) bool {
	reinstallChange := false
	if d.HasChange("ssh_keys") {
		sshKeys := d.Get("ssh_keys")
		if sshKeys != nil {
			reinstallChange = true
		}
	}

	if d.HasChange("root_password") {
		rootPassword := d.Get("root_password")
		if rootPassword != nil {
			reinstallChange = true
		}
	}

	if d.HasChange("user_data") {
		userData := d.Get("user_data").(string)
		if userData != "" {
			reinstallChange = true
		}
	}
	if d.HasChange("image_id") {
		imageId := d.Get("image_id").(string)
		if imageId != "" {
			reinstallChange = true
		}
	}
	if d.HasChange("default_user") {
		defaultUser := d.Get("default_user").(string)
		if defaultUser != "" {
			reinstallChange = true
		}
	}

	return reinstallChange
}

func updateInstanceValues(d *schema.ResourceData, client *openapi.APIClient, ctx context.Context, instanceId int64, diags diag.Diagnostics, m interface{}) diag.Diagnostics {

	patchInstanceRequest := *openapi.NewPatchInstanceRequestWithDefaults()
	displayName := d.Get("display_name").(string)
	patchInstanceRequest.DisplayName = &displayName

	res, httpResp, err := client.InstancesApi.
		PatchInstance(context.Background(), instanceId).
		PatchInstanceRequest(patchInstanceRequest).
		XRequestId(uuid.NewV4().String()).Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}
	d.SetId(strconv.Itoa(int(res.Data[0].InstanceId)))
	return diags
}

func reinstall(d *schema.ResourceData, client *openapi.APIClient, ctx context.Context, instanceId int64, diags diag.Diagnostics, m interface{}) diag.Diagnostics {
	patchInstanceRequest := buildReinstallRequest(d)

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
	return diags
}

func resourceInstanceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	if pnIds, ok := d.GetOk("private_network_ids"); ok {
		pnSet := pnIds.(*schema.Set).List()
		for _, id := range pnSet {
			privateNetworkId := int64(id.(int))
			if privateNetworkId != 0 {
				_, _ = unassignInstanceToPrivateNetwork(diags, client, privateNetworkId, instanceId)
			}
		}
	}

	_, httpResp, err := client.InstancesApi.
		CancelInstance(ctx, instanceId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")
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
	ipConfig := buildIpConfig(&instance.IpConfig)
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
	addOns := buildAddons(instance.AddOns)
	if err := d.Set("add_ons", addOns); err != nil && len(addOns) > 0 {
		return diag.FromErr(err)
	}
	if err := d.Set("error_message", instance.ErrorMessage); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("product_type", instance.ProductType); err != nil {
		return diag.FromErr(err)
	}
	additionalIps := buildAdditionalIps(instance.AdditionalIps)
	if err := d.Set("additional_ips", additionalIps); err != nil &&
		len(additionalIps) > 0 {
		return diag.FromErr(err)
	}

	return diags
}

func buildIpConfig(ipConfigResponse *openapi.IpConfig2) []interface{} {
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

func buildAddons(addOnResponse []openapi.AddOnResponse) []map[string]interface{} {
	if addOnResponse != nil {
		var addOns []map[string]interface{}
		for _, addOn := range addOnResponse {
			builtAddOn := make(map[string]interface{})
			builtAddOn["id"] = strconv.FormatInt(addOn.Id, 10)
			builtAddOn["quantity"] = addOn.Quantity

			addOns = append(addOns, builtAddOn)
		}
		return addOns
	}
	return nil
}

func buildAdditionalIps(
	additionalIpsResponse []openapi.AdditionalIp,
) []map[string]interface{} {
	if additionalIpsResponse != nil {
		additionalIps := []map[string]interface{}{}

		for _, ipV4 := range additionalIpsResponse {
			v4 := make(map[string]interface{})
			ipConfig := make(map[string]interface{})
			ipConfig["ip"] = ipV4.V4.Ip
			ipConfig["netmask_cidr"] = ipV4.V4.NetmaskCidr
			ipConfig["gateway"] = ipV4.V4.Gateway

			v4["v4"] = []interface{}{ipConfig}
			additionalIps = append(additionalIps, v4)
		}

		return additionalIps
	}

	return nil
}

func pollInstanceInstalled(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	ctx context.Context,
	instanceId int64,
) (*openapi.InstanceResponse, diag.Diagnostics) {
	res, httpResp, err := client.InstancesApi.
		RetrieveInstance(ctx, instanceId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return nil, HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return nil, MultipleDataObjectsError(diags)
	}

	status := res.Data[0].Status

	if status == openapi.PROVISIONING || status == openapi.INSTALLING {
		time.Sleep(time.Second)
		return pollInstanceInstalled(diags, client, ctx, instanceId)
	}

	return &res.Data[0], nil
}

func syncPrivateNetworksStatus(
	ctx context.Context,
	diags diag.Diagnostics,
	client *openapi.APIClient,
	d *schema.ResourceData,
	privateNetworkIds []int64,
	instanceId int64,
) diag.Diagnostics {
	// 1. Ensure instance is running / ready before checking or modifying private networks
	_, pollDiag := pollInstanceInstalled(diags, client, ctx, instanceId)
	if len(pollDiag) > 0 {
		return pollDiag
	}

	needsReinstall := false
	needsRestart := false

	for _, privateNetworkId := range privateNetworkIds {
		// Fetch current status of the instance in this private network
		status, err := getInstancePrivateNetworkStatus(ctx, client, privateNetworkId, instanceId)
		if err != nil {
			return diag.FromErr(err)
		}

		// If instance is not in the private network, add addon and assign
		if status == "not_found" {
			httpResp, err := retryAddPrivateNetworkAddOnToInstance(diags, client, instanceId, 0)
			if err != nil && !strings.Contains(err.Error(), httpConflict) {
				return HandleResponseErrors(diags, httpResp)
			}

			httpResp, err = assignInstanceToPrivateNetwork(diags, client, privateNetworkId, instanceId)
			if err != nil {
				return HandleResponseErrors(diags, httpResp)
			}

			// Re-fetch status after assigning
			status, err = getInstancePrivateNetworkStatus(ctx, client, privateNetworkId, instanceId)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		switch status {
		case "ok":
			// already fine
		case "reinstall":
			needsReinstall = true
		case "restart":
			needsRestart = true
		case "installing":
			_, pollDiag = pollInstanceInstalled(diags, client, ctx, instanceId)
			if len(pollDiag) > 0 {
				return pollDiag
			}
		case "reinstallation failed":
			return diag.Errorf("Contabo instance %d private network reinstallation failed in network %d", instanceId, privateNetworkId)
		}
	}

	// If at least one network requires reinstall, reinstall takes precedence and provisions NICs
	if needsReinstall {
		reinstallReq := buildReinstallRequest(d)

		_, httpResp, err := client.InstancesApi.
			ReinstallInstance(ctx, instanceId).
			XRequestId(uuid.NewV4().String()).
			ReinstallInstanceRequest(*reinstallReq).
			Execute()
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}

		// Wait for reinstall to complete
		_, pollDiag = pollInstanceInstalled(diags, client, ctx, instanceId)
		if len(pollDiag) > 0 {
			return pollDiag
		}
	} else if needsRestart {
		_, httpResp, err := client.InstanceActionsApi.
			Restart(ctx, instanceId).
			XRequestId(uuid.NewV4().String()).
			Execute()
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}

		_, pollDiag = pollInstanceInstalled(diags, client, ctx, instanceId)
		if len(pollDiag) > 0 {
			return pollDiag
		}
	}

	return nil
}

func getInstancePrivateNetworkStatus(
	ctx context.Context,
	client *openapi.APIClient,
	privateNetworkId int64,
	instanceId int64,
) (string, error) {
	pnRes, _, err := client.PrivateNetworksApi.
		RetrievePrivateNetwork(ctx, privateNetworkId).
		XRequestId(uuid.NewV4().String()).
		Execute()
	if err != nil {
		return "", err
	}
	if len(pnRes.Data) == 0 {
		return "not_found", nil
	}

	for _, inst := range pnRes.Data[0].Instances {
		if inst.InstanceId == instanceId {
			return inst.Status, nil
		}
	}
	return "not_found", nil
}

func buildReinstallRequest(d *schema.ResourceData) *openapi.ReinstallInstanceRequest {
	req := openapi.NewReinstallInstanceRequestWithDefaults()

	if imageId := d.Get("image_id").(string); imageId != "" {
		req.ImageId = imageId
	}
	if userData := d.Get("user_data").(string); userData != "" {
		req.UserData = &userData
	}
	if defaultUser := d.Get("default_user").(string); defaultUser != "" {
		req.DefaultUser = &defaultUser
	}
	if rootPassword, ok := d.GetOk("root_password"); ok {
		rootPassword64 := int64(rootPassword.(int))
		req.RootPassword = &rootPassword64
	}
	if sshKeys, ok := d.GetOk("ssh_keys"); ok {
		var sshKeys64 []int64
		for _, key := range sshKeys.([]interface{}) {
			sshKeys64 = append(sshKeys64, int64(key.(int)))
		}
		req.SshKeys = &sshKeys64
	}

	return req
}


