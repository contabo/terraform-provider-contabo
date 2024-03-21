package contabo

import (
	"context"

	"strconv"
	"strings"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func resourceTagAssignment() *schema.Resource {
	return &schema.Resource{
		Description:   "Tag assignment marks the specified resource with the specified tag for organizing purposes or to restrict access to that resource.",
		CreateContext: resourceTagAssignmentCreate,
		ReadContext:   resourceTagAssignmentRead,
		DeleteContext: resourceTagAssignmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
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

func generateTagAssignmentId(tagIdString string, resourceType string, resourceId string) string {
	return tagIdString + "_" + resourceType + "_" + resourceId
}

func getTagIdFromId(tagAssignmentId string) int64 {
	tagIdString := strings.Split(tagAssignmentId, "_")[0]
	var tagId int64
	if tagIdString != "" {
		tagId, _ = strconv.ParseInt(tagIdString, 10, 64)
	}
	return tagId
}

func getResourceTypeFromId(tagAssignmentId string) string {
	resourceType := strings.Split(tagAssignmentId, "_")[1]
	return resourceType
}

func getResourceIdFromId(tagAssignmentId string) string {
	resourceId := strings.Split(tagAssignmentId, "_")[2]
	return resourceId
}

func resourceTagAssignmentCreate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	tagId := d.Get("tag_id").(int)
	resourceType := d.Get("resource_type").(string)
	resourceId := d.Get("resource_id").(string)

	_, httpResp, err := client.TagAssignmentsApi.
		CreateAssignment(context.Background(), int64(tagId), resourceType, resourceId).
		XRequestId(uuid.NewV4().String()).Execute()
	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}
	id := generateTagAssignmentId(strconv.Itoa(int(tagId)), resourceType, resourceId)
	d.SetId(id)
	return resourceTagAssignmentRead(ctx, d, m)
}

func resourceTagAssignmentRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	tagId := getTagIdFromId(d.Id())
	resourceType := getResourceTypeFromId(d.Id())
	resourceId := getResourceIdFromId(d.Id())

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

	return AddTagAssignmentToData(res.Data[0], d, diags)
}

func resourceTagAssignmentDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	tagId := getTagIdFromId(d.Id())
	resourceType := getResourceTypeFromId(d.Id())
	resourceId := getResourceIdFromId(d.Id())

	httpResp, err := client.TagAssignmentsApi.
		DeleteAssignment(context.Background(), tagId, resourceType, resourceId).
		XRequestId(uuid.NewV4().String()).Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")

	return diags
}

func AddTagAssignmentToData(
	tagAssignment openapi.AssignmentResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {

	id := generateTagAssignmentId(strconv.Itoa(int(tagAssignment.TagId)), tagAssignment.ResourceType, tagAssignment.ResourceId)
	if err := d.Set("id", id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tag_id", tagAssignment.TagId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tag_name", tagAssignment.TagName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_id", tagAssignment.ResourceId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_type", tagAssignment.ResourceType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_name", tagAssignment.ResourceName); err != nil {
		return diag.FromErr(err)
	}
	return diags
}
