package contabo

import (
	"context"
	"strconv"

	apiClient "contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourceInstance() *schema.Resource {
	return &schema.Resource{
		Description: "The Compute Management API allows you to manage compute resources (e.g. creation, deletion, starting, stopping) as well as managing snapshots and custom images. It also supports [cloud-init](https://cloud-init.io/) at least on our default images (for custom images you will need to provide cloud-init support packages). The API offers providing cloud-init scripts via the user_data field. Custom images must be provided in .qcow2 or .iso format.",
		ReadContext: dataSourceInstanceRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
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
			"created_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The creation date of the compute instance.",
			},
			"cancel_date": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
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
				Computed:    true,
				Description: "Cloud-Init Config in order to customize during start of compute instance.",
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
				Description: "Default user name created for login during (re-)installation with administrative privileges. Allowed values for Linux/BSD are admin (use sudo to apply administrative privileges like root) or root. Allowed values for Windows are admin (has administrative privileges like administrator) or administrator.See our [api documentation](https://api.contabo.com/#tag/Instances/operation/createInstance) for available default users.",
			},
			"period": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Initial contract period in months. Available periods are: 1, 3, 6 and 12 months. The default setting is 1 month.",
			},
			"additional_ips_v4": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "All other additional IP addresses of the instance.",
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
	}
}

func dataSourceInstanceRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*apiClient.APIClient)

	var instanceId int64
	var err error
	id := d.Get("id").(string)
	if id != "" {
		instanceId, err = strconv.ParseInt(id, 10, 64)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.InstancesApi.
		RetrieveInstance(ctx, int64(instanceId)).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(strconv.Itoa(int(res.Data[0].InstanceId)))

	return AddInstanceToData(
		res.Data[0],
		d,
		diags,
	)
}
