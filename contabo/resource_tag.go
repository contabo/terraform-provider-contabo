package contabo

import (
	"context"
	"strconv"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func resourceTag() *schema.Resource {
	return &schema.Resource{
		Description:   "Tags are Customer-defined labels which can be attached to any resource in your account. Tag API represent simple CRUD functions and allow you to manage your tags. Use tags to group your resources. For example you can define some user group with tag and give them permission to create compute instances.",
		CreateContext: resourceTagCreate,
		ReadContext:   resourceTagRead,
		UpdateContext: resourceTagUpdate,
		DeleteContext: resourceTagDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the tag. Use it to manage it!",
			},
			"color": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The tag color.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The tag name.",
			},
		},
	}
}

func resourceTagCreate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	tagName := d.Get("name").(string)
	tagColor := d.Get("color").(string)

	createTagRequest := openapi.NewCreateTagRequestWithDefaults()
	createTagRequest.Name = tagName
	createTagRequest.Color = tagColor

	res, httpResp, err := client.TagsApi.
		CreateTag(context.Background()).
		XRequestId(uuid.NewV4().String()).
		CreateTagRequest(*createTagRequest).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	if len(res.Data) == 0 {
		return NoDataError(diags)
	} else if len(res.Data) > 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(strconv.Itoa(int(res.Data[0].TagId)))
	return resourceTagRead(ctx, d, m)
}

func resourceTagRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	tagId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.TagsApi.
		RetrieveTag(ctx, tagId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	if len(res.Data) != 1 {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Internal Error: should have returned only one object",
		})
	}

	return AddTagToData(res.Data[0], d, diags)
}

func resourceTagUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	tagId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	updateTagRequest := openapi.NewUpdateTagRequestWithDefaults()
	anyChange := false

	if d.HasChange("name") {
		tagName := d.Get("name").(string)
		updateTagRequest.Name = &tagName
		anyChange = true
	}

	if d.HasChange("color") {
		tagColor := d.Get("color").(string)
		updateTagRequest.Color = &tagColor
		anyChange = true
	}

	if anyChange {
		_, httpResp, err := client.TagsApi.
			UpdateTag(context.Background(), tagId).
			XRequestId(uuid.NewV4().String()).
			UpdateTagRequest(*updateTagRequest).
			Execute()

		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}
		return resourceTagRead(ctx, d, m)
	}

	return diags
}

func resourceTagDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	tagId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	httpResp, err := client.TagsApi.
		DeleteTag(ctx, tagId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")

	return diags
}

func AddTagToData(
	tag openapi.TagResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	id := strconv.Itoa(int(tag.TagId))
	if err := d.Set("id", id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", tag.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("color", tag.Color); err != nil {
		return diag.FromErr(err)
	}
	return diags
}
