package contabo

import (
	"context"

	apiClient "contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourceImage() *schema.Resource {
	return &schema.Resource{
		Description: "In order to provide a custom image, please specify an URL from which the image can be downloaded directly. A custom image must be in either `.iso` or `.qcow2` format. Other formats will be rejected. Please note that downloading can take a while depending on network speed resp. bandwidth and size of image. You can check the status by retrieving information about the image via a GET request. Download will be rejected if you have exceeded your limits.",
		ReadContext: dataSourceImageRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the image. Use it to manage it!",
			},
			"last_updated": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Time of the last update of the image.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the image.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "Description of the image.",
			},
			"uploaded_size_mb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the uploaded image in megabyte.",
			},
			"os_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Provided type of operating system (OS). Please specify Windows for MS `Windows` and `Linux` for other OS. Specifying wrong OS type may lead to disfunctional cloud instance.",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Version number to distinguish the contents of an image e.g. the version of the operating system.",
			},
			"format": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Format of your image `iso` or `qcow`.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Downloading status of the image (`downloading`, `downloaded` or `error`).",
			},
			"error_message": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "If the image is in an error state (see status property), the error message can be seen in this field.",
			},
			"standard_image": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Flag indicating that the image is either a standard (true) or a custom image (false).",
			},
			"creation_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The creation date of the image.",
			},
		},
	}
}

func dataSourceImageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*apiClient.APIClient)

	imageId := d.Get("id").(string)

	if imageId == "" {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "imageId should not be empty",
		})
	}

	res, httpResp, err := client.ImagesApi.
		RetrieveImage(ctx, imageId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(res.Data[0].ImageId)

	return AddImageToData(res.Data[0], d, diags)
}
