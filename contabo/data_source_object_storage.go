package contabo

import (
	"context"

	apiClient "contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourceObjectStorage() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceObjectStorageRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cancel_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"s3_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"s3_tenant_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auto_scaling": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
						"size_limit_tb": {
							Type:     schema.TypeFloat,
							Computed: true,
							Optional: true,
						},
						"error_message": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
					},
				},
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"customer_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"data_center": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"total_purchased_space_tb": {
				Type:     schema.TypeFloat,
				Computed: true,
			},
		},
	}
}

func dataSourceObjectStorageRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*apiClient.APIClient)

	var objectStorageId string
	var err error
	id := d.Get("id").(string)
	if id != "" {
		objectStorageId = id
	}

	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.ObjectStoragesApi.RetrieveObjectStorage(ctx, objectStorageId).XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(res.Data[0].ObjectStorageId)

	return AddObjectStorageToData(
		res.Data[0],
		d,
		diags,
	)
}
