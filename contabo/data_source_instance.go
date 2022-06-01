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
		ReadContext: dataSourceInstanceRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
				Computed: true,
				Type:     schema.TypeInt,
			},
			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cancel_date": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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
				Computed: true,
			},
			"license": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"period": {
				Type:     schema.TypeInt,
				Computed: true,
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
