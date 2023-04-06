package contabo

import (
	"context"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

const (
	DOWNLOADING string = "downloading"
	ERROR string = "error"
)

func resourceImage() *schema.Resource {
	return &schema.Resource{
		Description:   "In order to provide a custom image, please specify an URL from which the image can be downloaded directly. A custom image must be in either `.iso` or `.qcow2` format. Other formats will be rejected. Please note that downloading can take a while depending on network speed resp. bandwidth and size of image. You can check the status by retrieving information about the image via a GET request. Download will be rejected if you have exceeded your limits.",
		CreateContext: resourceImageCreate,
		ReadContext:   resourceImageRead,
		UpdateContext: resourceImageUpdate,
		DeleteContext: resourceImageDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the image. Use it to manage it!",
			},
			"last_updated": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Time of the last update of the image.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the image.",
			},
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Description of the image.",
			},
			"image_url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "URL from which the image has been downloaded.",
				ForceNew:    true,
			},
			"uploaded_size_mb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the uploaded image in megabyte.",
			},
			"os_type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Provided type of operating system (OS). Please specify Windows for MS `Windows` and `Linux` for other OS. Specifying wrong OS type may lead to disfunctional cloud instance.",
			},
			"version": {
				Type:        schema.TypeString,
				Required:    true,
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

	image, diag := pollImageDownloaded(diags, client, ctx, imageId)

	if err != nil || image == nil {
		diags = append(diags, diag...)
		return AddImageToData(res.Data[0], d, diags)
	}

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

func pollImageDownloaded(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	ctx context.Context,
	imageId string,
) (*openapi.ImageResponse, diag.Diagnostics) {
	res, httpResp, err := client.ImagesApi.
		RetrieveImage(ctx, imageId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return nil, HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return nil, MultipleDataObjectsError(diags)
	}

	status := res.Data[0].Status

	if status == ERROR {
		return nil, HandleDownloadErrors(diags)
	}

	if status == DOWNLOADING {
		time.Sleep(time.Second)
		return pollImageDownloaded(diags, client, ctx, imageId)
	}

	return &res.Data[0], nil
}

