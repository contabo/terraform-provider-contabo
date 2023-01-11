package contabo

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func resourceObjectStorage() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage S3 compatible Object Storage. With the Object Storage API you can create Object Storages in different locations. Please note that you can only have one Object Storage per location. Furthermore, you can increase the amount of storage space and control the autoscaling feature which allows you to automatically perform a monthly upgrade of the disk space to the specified maximum. You might also inspect the usage. This API is not the S3 API itself. For accessing the S3 API directly or with S3 compatible tools like `aws` cli and after having created / upgraded your Object Storage please use the S3 URL from this Storage API and refer to the User Mangement API to retrieve the S3 credentials.",
		CreateContext: resourceObjectStorageCreate,
		ReadContext:   resourceObjectStorageRead,
		UpdateContext: resourceObjectStorageUpgrade,
		DeleteContext: resourceObjectStorageCancel,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
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
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Status of this object storage.  It can be set to `enabled`, `disabled` or `error`.",
						},
						"size_limit_tb": {
							Type:        schema.TypeFloat,
							Optional:    true,
							Description: "Autoscaling size limit for the current object storage.",
						},
						"error_message": {
							Type:        schema.TypeString,
							Optional:    true,
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
				Required:    true,
				Description: "Region where the Object Storage should be located. Default region is the EU. Following regions are available: `EU`,`US-central`, `SIN`.",
			},
			"total_purchased_space_tb": {
				Type:        schema.TypeFloat,
				Required:    true,
				Description: "Amount of purchased / requested object storage in terabyte.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Required:    false,
				Optional:    true,
				Computed:    true,
				Description: "Display name for object storage.",
			},
		},
	}
}

func resourceObjectStorageCreate(
	ctx context.Context,
	data *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	var err error

	client := m.(*openapi.APIClient)

	objectStorageRegion := data.Get("region").(string)
	objectStorageTotalPurchasedSpaceTB := data.Get("total_purchased_space_tb").(float64)
	objectStorageAutoScaling, err := TryFlattenSliceOfSingleMap(data.Get("auto_scaling"))

	if err != nil {
		return diag.FromErr(err)
	}

	createObjectStorageRequest := openapi.NewCreateObjectStorageRequestWithDefaults()
	createObjectStorageRequest.TotalPurchasedSpaceTB = objectStorageTotalPurchasedSpaceTB
	createObjectStorageRequest.Region = objectStorageRegion

	if objectStorageAutoScaling != nil {
		autoScalingState := objectStorageAutoScaling["state"].(string)
		autoScalingLimit := objectStorageAutoScaling["size_limit_tb"].(float64)

		autoScaling := openapi.AutoScalingTypeRequest{
			State:       autoScalingState,
			SizeLimitTB: autoScalingLimit,
		}
		createObjectStorageRequest.AutoScaling = &autoScaling
	}

	displayName := data.Get("display_name").(string)

	if displayName != "" {
		createObjectStorageRequest.DisplayName = &displayName
	}

	res, httpResp, err := client.ObjectStoragesApi.
		CreateObjectStorage(ctx).
		XRequestId(uuid.NewV4().String()).
		CreateObjectStorageRequest(*createObjectStorageRequest).
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

	data.SetId(res.Data[0].ObjectStorageId)

	return resourceObjectStorageRead(ctx, data, m)
}

func resourceObjectStorageRead(
	ctx context.Context,
	data *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	objectStorageId := data.Id()

	res, httpResp, err := client.
		ObjectStoragesApi.
		RetrieveObjectStorage(ctx, objectStorageId).
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

	return AddObjectStorageToData(res.Data[0], data, diags)
}

func resourceObjectStorageUpgrade(
	ctx context.Context,
	data *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	doUpgrade := false

	objectStorageId := data.Id()

	upgradeObjectStoragaRequest := openapi.NewUpgradeObjectStorageRequest()

	if data.HasChange("total_purchased_space_tb") {
		newTotalPurchasedSpace := data.Get("total_purchased_space_tb").(float64)
		upgradeObjectStoragaRequest.TotalPurchasedSpaceTB = &newTotalPurchasedSpace
		doUpgrade = true
	}

	if data.HasChange("auto_scaling") {
		objectStorageAutoScaling, err := TryFlattenSliceOfSingleMap(data.Get("auto_scaling"))

		if err != nil {
			return diag.FromErr(err)
		}

		if objectStorageAutoScaling != nil {
			autoScalingState := objectStorageAutoScaling["state"].(string)
			autoScalingLimit := objectStorageAutoScaling["size_limit_tb"].(float64)

			autoScaling := openapi.UpgradeAutoScalingType{}

			if autoScalingState != "" && autoScalingLimit != 0 {
				autoScaling = openapi.UpgradeAutoScalingType{
					State:       &autoScalingState,
					SizeLimitTB: &autoScalingLimit,
				}
			} else if autoScalingState != "" && autoScalingLimit == 0 {
				autoScaling = openapi.UpgradeAutoScalingType{
					State: &autoScalingState,
				}
			} else if autoScalingState == "" && autoScalingLimit != 0 {
				autoScaling = openapi.UpgradeAutoScalingType{
					SizeLimitTB: &autoScalingLimit,
				}
			}

			upgradeObjectStoragaRequest.AutoScaling = &autoScaling
			doUpgrade = true
		}
	}

	if doUpgrade {
		_, httpResp, err := client.ObjectStoragesApi.
			UpgradeObjectStorage(ctx, objectStorageId).
			XRequestId(uuid.NewV4().String()).
			UpgradeObjectStorageRequest(*upgradeObjectStoragaRequest).
			Execute()
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}

		data.Set("last_updated", time.Now().Format(time.RFC850))
	}

	if data.HasChange("display_name") {
		displayName := data.Get("display_name").(string)
		patchObjectStorageRequest := openapi.NewPatchObjectStorageRequest(displayName)
		_, httpResp, err := client.ObjectStoragesApi.UpdateObjectStorage(ctx, objectStorageId).
			XRequestId(uuid.NewV4().String()).
			PatchObjectStorageRequest(*patchObjectStorageRequest).
			Execute()

		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}

		data.Set("last_updated", time.Now().Format(time.RFC850))
	}

	return resourceObjectStorageRead(ctx, data, m)
}

func resourceObjectStorageCancel(
	ctx context.Context,
	data *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	objectStorageId := data.Id()

	_, httpResp, err := client.ObjectStoragesApi.
		CancelObjectStorage(ctx, objectStorageId).
		XRequestId(uuid.NewV4().String()).
		Execute()
	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	data.SetId("")

	return diags
}

func AddObjectStorageToData(
	objectStorage openapi.ObjectStorageResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	id := objectStorage.ObjectStorageId
	if err := d.Set("id", id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("created_date", objectStorage.CreatedDate.Format(time.RFC850)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cancel_date", objectStorage.CancelDate); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tenant_id", objectStorage.TenantId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("customer_id", objectStorage.CustomerId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("s3_url", objectStorage.S3Url); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("s3_tenant_id", objectStorage.S3TenantId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", objectStorage.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("data_center", objectStorage.DataCenter); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region", objectStorage.Region); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("total_purchased_space_tb", objectStorage.TotalPurchasedSpaceTB); err != nil {
		return diag.FromErr(err)
	}
	autoScaling := BuildAutoScaling(&objectStorage.AutoScaling)
	if err := d.Set("auto_scaling", autoScaling); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("display_name", objectStorage.DisplayName); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func BuildAutoScaling(autoScalingResponse *openapi.AutoScalingTypeResponse) interface{} {
	if autoScalingResponse != nil {
		autoScaling := make(map[string]interface{})
		autoScaling["state"] = autoScalingResponse.State
		autoScaling["size_limit_tb"] = autoScalingResponse.SizeLimitTB
		autoScaling["error_message"] = autoScalingResponse.ErrorMessage

		return []interface{}{autoScaling}
	}

	return nil
}

// Attention! returns `nil` if input is `nil`
func TryFlattenSliceOfSingleMap(obj interface{}) (map[string]interface{}, error) {
	if obj == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(obj)

	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("[TryFlattenSliceOfSingleMap] provided value '%v' was not a slice nor map", obj)
	}

	if rv.Len() == 0 {
		return nil, nil
	} else if rv.Len() > 1 {
		return nil, fmt.Errorf("[TryFlattenSliceOfSingleMap] provided slice '%v' has not exacly one item", obj)
	}

	maybeMap := reflect.ValueOf(rv.Index(0).Interface())

	if maybeMap.Kind() != reflect.Map {
		return nil, fmt.Errorf("[TryFlattenSliceOfSingleMap] the item in provided slice '%v' was not a map[string]interface{}", obj)
	}

	var out = make(map[string]interface{})

	for _, key := range maybeMap.MapKeys() {
		strct := maybeMap.MapIndex(key)
		out[key.String()] = strct.Interface()
	}

	return out, nil

}
