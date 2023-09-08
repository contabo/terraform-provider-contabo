package contabo

import (
	"context"
	"strconv"

	apiClient "contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourcePrivateNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a Contabo [Private Network](https://api.contabo.com/#tag/Private-Networks) data source.  Private Networks can contain your compute instances whereby they are able to communicate with each other in full usolation, using private IP addresses ",
		ReadContext: dataSourcePrivateNetworkRead,
		Schema: map[string]*schema.Schema{
			"created_date": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The creation date of the Private Network.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Time of the last update of the private network.",
			},
			"id": {
				Type:        schema.TypeString,
				Description: "The identifier of the Private Network. Use it to manage it!",
				Required:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the Private Network. It may contain letters, numbers, colons, dashes, and underscores. There is a limit of 255 characters per Private Network name.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Private Network. There is a limit of 255 characters per Private Network.",
			},
			"instance_ids": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Optional:    true,
				Description: "Add the instace Ids to the private network here. If you do not add any instance Ids an empty private network will be created.",
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The identifier of the compute instance.",
						},
						"display_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The instance name chosen by the customer that will be shown in the customer panel.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the compute instance.",
						},
						"private_ip_config": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of all private IP addresses of the compute instance.",
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
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "State of the instance in the Private Network. The status can be one of 'ok', 'restart', 'reinstall', 'reinstallation failed', 'installing'",
						},
						"error_message": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "If the instance is in an error state (see status property), the error message can be seen in this field.",
						},
					},
				},
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "EU",
				Description: "The region where the Private Network should be located. Default region is the EU.",
			},
			"region_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the region where the Private Network is located.",
			},
			"data_center": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The specific data center where the Private Network is located.",
			},
			"available_ips": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The totality of available IPs in the Private Network.",
			},
			"cidr": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The cidr range of the Private Network.",
			},
		},
	}
}

func dataSourcePrivateNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*apiClient.APIClient)

	var privateNetworktId int64
	var err error
	id := d.Get("id").(string)
	if id != "" {
		privateNetworktId, err = strconv.ParseInt(id, 10, 64)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.PrivateNetworksApi.
		RetrievePrivateNetwork(ctx, privateNetworktId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(strconv.Itoa(int(res.Data[0].PrivateNetworkId)))

	return AddPrivateNetworkToData(res.Data[0], d, diags)
}
