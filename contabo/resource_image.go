package contabo

import (
	"context"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func resourceImage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceImageCreate,
		ReadContext:   resourceImageRead,
		UpdateContext: resourceImageUpdate,
		DeleteContext: resourceImageDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_updated": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"image_url": {
				Type: schema.TypeString,
				Required: true,
			},
			"uploaded_size_mb": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"os_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeString,
				Required: true,
			},
			"format": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"error_message": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"standard_image": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceImageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	createImageRequest := openapi.NewCreateCustomImageRequestWithDefaults()

	name := d.Get("name").(string)
	description := d.Get("description")
	osType := d.Get("os_type").(string)
	version := d.Get("version").(string)
	imageUrl := d.Get("image_url").(string)

	if name != "" {
		createImageRequest.Name = name
	}
	if description != nil {
		gotDescritpion := description.(string)
		createImageRequest.Description = &gotDescritpion
	}
	if osType != "" {
		createImageRequest.OsType = osType
	}
	if version != "" {
		createImageRequest.Version = version
	}
	if imageUrl != "" {
		createImageRequest.Url = imageUrl
	}

	res, httpResp, err := client.ImagesApi.
		CreateCustomImage(ctx).
		XRequestId(uuid.NewV4().String()).
		CreateCustomImageRequest(*createImageRequest).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(res.Data[0].ImageId)

	return resourceImageRead(ctx, d, m)
}

func resourceImageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	imageId := d.Id()

	res, httpResp, err := client.ImagesApi.
		RetrieveImage(ctx, imageId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	return AddImageToData(res.Data[0], d, diags)
}

func resourceImageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	anyChange := false
	imageId := d.Id()

	updateImageRequest := openapi.NewUpdateCustomImageRequest()

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		updateImageRequest.Name = &newName
		anyChange = true
	}

	if d.HasChange("description") {
		newDescription := d.Get("description").(string)
		updateImageRequest.Description = &newDescription
		anyChange = true
	}

	if anyChange {
		res, httpResp, err := client.ImagesApi.
			UpdateImage(ctx, imageId).
			XRequestId(uuid.NewV4().String()).
			UpdateCustomImageRequest(*updateImageRequest).
			Execute()

		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		} else if len(res.Data) != 1 {
			return MultipleDataObjectsError(diags)
		}

		d.SetId(imageId)

		return resourceImageRead(ctx, d, m)
	}

	return diags
}

func resourceImageDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	imageId := d.Id()

	httpResp, err := client.ImagesApi.
		DeleteImage(ctx, imageId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")

	return diags
}

func AddImageToData(
	image openapi.ImageResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	if err := d.Set("id", image.ImageId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", image.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", image.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("uploaded_size_mb", image.UploadedSizeMb); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("os_type", image.OsType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("version", image.Version); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("format", image.Format); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", image.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("error_message", image.ErrorMessage); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("standard_image", image.StandardImage); err != nil {
		return diag.FromErr(err)
	}
	creationDate := image.CreationDate.Format(time.RFC850)
	if err := d.Set("creation_date", creationDate); err != nil {
		return diag.FromErr(err)
	}

	return diags
}