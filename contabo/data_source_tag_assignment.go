package contabo

import (
	"context"

	apiClient "contabo.com/openapi"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourceTagAssignment() *schema.Resource {
	return &schema.Resource{
		Description: "Tag assignment marks the specified resource with the specified tag for organizing purposes or to restrict access to that resource.",
		ReadContext: dataSourceTagAssignmentRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the tag assignment. Use it to manage it!",
			},
			"tag_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Optional:    true,
				Description: "The identifier of the tag.",
			},
			"tag_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the tag.",
			},
			"resource_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The resource type.",
			},
			"resource_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The resource id.",
			},
			"resource_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The resource name.",
			},
		},
	}
}

func dataSourceTagAssignmentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*apiClient.APIClient)

	id := d.Get("id").(string)
	tagId := getTagIdFromId(id)
	resourceType := getResourceTypeFromId(id)
	resourceId := getResourceIdFromId(id)

	res, httpResp, err := client.TagAssignmentsApi.
		RetrieveAssignment(context.Background(), tagId, resourceType, resourceId).
		XRequestId(uuid.NewV4().String()).Execute()
	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	if len(res.Data) == 0 {
		return NoDataError(diags)
	} else if len(res.Data) > 1 {
		return MultipleDataObjectsError(diags)
	}
	d.SetId(id)
	return AddTagAssignmentToData(res.Data[0], d, diags)

}
