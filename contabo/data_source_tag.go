package contabo

import (
	"context"
	"strconv"

	apiClient "contabo.com/openapi"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourceTag() *schema.Resource {
	return &schema.Resource{
		Description: "Tags are Customer-defined labels which can be attached to any resource in your account. Tag API represent simple CRUD functions and allow you to manage your tags. Use tags to group your resources. For example you can define some user group with tag and give them permission to create compute instance.",
		ReadContext: dataSourceTagRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the tag. Use it to manage it!",
			},
			"color": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The tag color.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The tag name.",
			},
		},
	}
}

func dataSourceTagRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*apiClient.APIClient)

	var tagId int64
	var err error
	id := d.Get("id").(string)
	if id != "" {
		tagId, err = strconv.ParseInt(id, 10, 64)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.TagsApi.
		RetrieveTag(ctx, tagId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(strconv.Itoa(int(res.Data[0].TagId)))

	return AddTagToData(res.Data[0], d, diags)
}
