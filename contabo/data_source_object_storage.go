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
		Description: "Manage S3 compatible Object Storage. With the Object Storage API you can create Object Storages in different locations. Please note that you can only have one Object Storage per location. Furthermore, you can increase the amount of storage space and control the autoscaling feature which allows you to automatically perform a monthly upgrade of the disk space to the specified maximum. You might also inspect the usage. This API is not the S3 API itself. For accessing the S3 API directly or with S3 compatible tools like `aws` cli and after having created / upgraded your Object Storage please use the S3 URL from this Storage API and refer to the User Mangement API to retrieve the S3 credentials.",
		ReadContext: dataSourceObjectStorageRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional: true,
				Description: "The identifier of the Object Storage. Use it to manage it!",
			},
			"created_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The creation date of the Object Storage.",
			},
			"cancel_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date on which the Object Storage will be cancelled and therefore no longer available.",
			},
			"s3_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "S3 URL to connect to your S3 compatible Object Storage.",
			},
			"s3_tenant_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Your S3 tenant Id. Only required for public sharing.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The object storage status. It can be set to `PROVISIONING`,`READY`,`UPGRADING`,`CANCELLED`,`ERROR` or `DISABLED`.",
			},
			"auto_scaling": {
				Type:     schema.TypeList,
				Computed:    true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional: true,
							Description: "Status of this object storage.  It can be set to `enabled`, `disabled` or `error`.",
						},
						"size_limit_tb": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Optional: true,
							Description: "Autoscaling size limit for the current object storage.",
						},
						"error_message": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional: true,
							Description: "If the autoscaling is in an error state (see status property), the error message can be seen in this field.",
						},
					},
				},
			},
			"tenant_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Your customer tenant Id.",
			},
			"customer_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Your customer number.",
			},
			"data_center": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Data center the object storage is located in.",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region where the Object Storage should be located. Default region is the EU. Following regions are available: `EU`,`US-central`, `SIN`.",
			},
			"total_purchased_space_tb": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Amount of purchased / requested object storage in terabyte.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Display name for object storage. Use it to manage it!",
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

	objectStorageId := d.Get("id").(string)
	objectStorageDisplayName := d.Get("display_name").(string)

	if objectStorageId == "" && objectStorageDisplayName == "" {
		return HandleMissingDataObjectsFilters(diags, "Missing required field", "You must provide either the `id` or `display_name` field.")
	} else if objectStorageId != "" && objectStorageDisplayName != "" {
		return HandleMissingDataObjectsFilters(diags, "Multiple filters provided", "You must provide only one of the following fields: `id` or `display_name`.")
	} else if (objectStorageId != "") {
		res, httpResp, err := client.ObjectStoragesApi.RetrieveObjectStorage(ctx, objectStorageId).XRequestId(uuid.NewV4().String()).Execute()

		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}

		if(len(res.Data) == 0) {
			return NoDataError(diags)
		}

		d.SetId(res.Data[0].ObjectStorageId)
		return AddObjectStorageToData(
			res.Data[0],
			d,
			diags,
		)
	} else if (objectStorageDisplayName != "") {
		res, httpResp, err := client.ObjectStoragesApi.RetrieveObjectStorageList(ctx).XRequestId(uuid.NewV4().String()).DisplayName(objectStorageDisplayName).Execute()

		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}

		if(len(res.Data) == 0) {
			return NoDataError(diags)
		} else if(len(res.Data) > 1) {
			return MultipleDataObjectsError(diags)
		}

		d.SetId(res.Data[0].ObjectStorageId)
		return AddObjectStorageToData(
			res.Data[0],
			d,
			diags,
		)
	}

	return nil
}
