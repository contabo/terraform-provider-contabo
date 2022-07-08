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
			"created_date": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The creation date of the Private Network.",
			},
			"updated_at": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Time of the last update of the private network.",
			},
			"id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the Private Network. Use it to manage it!",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the Private Network. It may contain letters, numbers, colons, dashes, and underscores. There is a limit of 255 characters per Private Network name.",
			},
			"description": &schema.Schema{
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
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "EU",
				Description: "The region where the Private Network should be located. Default region is the EU.",
			},
			"region_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the region where the Private Network is located.",
			},
			"data_center": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The specific data center where the Private Network is located.",
			},
			"available_ips": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The totality of available IPs in the Private Network.",
			},
			"cidr": &schema.Schema{
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
