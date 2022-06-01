package contabo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func resourceObjectStorage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceObjectStorageCreate,
		ReadContext:   resourceObjectStorageRead,
		UpdateContext: resourceObjectStorageUpgrade,
		DeleteContext: resourceObjectStorageCancel,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
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
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"size_limit_tb": {
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"error_message": {
							Type:     schema.TypeString,
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
				Required: true,
			},
			"total_purchased_space_tb": {
				Type:     schema.TypeFloat,
				Required: true,
			},
		},
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
	objectStorageAutoScaling, _ := StructToMap(data.Get("auto_scaling"))

	if err != nil {
		return diag.FromErr(err)
	}

	createObjectStorageRequest := openapi.NewCreateObjectStorageRequestWithDefaults()
	createObjectStorageRequest.TotalPurchasedSpaceTB = objectStorageTotalPurchasedSpaceTB
	createObjectStorageRequest.Region = objectStorageRegion

	if objectStorageAutoScaling != nil {
		autoScalingState := fmt.Sprintf("%v", objectStorageAutoScaling["state"])
		autoScalingLimit := fmt.Sprintf("%v", objectStorageAutoScaling["size_limit_tb"])
		autoScalingLimitFloat, err := strconv.ParseFloat(autoScalingLimit, 64)
		if err != nil {
			return diag.FromErr(err)
		}

		autoScaling := openapi.AutoScalingTypeRequest{
			State:       autoScalingState,
			SizeLimitTB: autoScalingLimitFloat,
		}
		createObjectStorageRequest.AutoScaling = &autoScaling
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
	anyChange := false

	objectStorageId := data.Id()

	upgradeObjectStoragaRequest := openapi.NewUpgradeObjectStorageRequest()

	if data.HasChange("total_purchased_space_tb") {
		newTotalPurchasedSpace := data.Get("total_purchased_space_tb").(float64)
		upgradeObjectStoragaRequest.TotalPurchasedSpaceTB = &newTotalPurchasedSpace
		anyChange = true
	}

	if data.HasChange("auto_scaling") {
		objectStorageAutoScaling, _ := StructToMap(data.Get("auto_scaling"))
		autoScalingState := fmt.Sprintf("%v", objectStorageAutoScaling["state"])
		autoScalingLimit := fmt.Sprintf("%v", objectStorageAutoScaling["size_limit_tb"])
		autoScalingLimitFloat, err := strconv.ParseFloat(autoScalingLimit, 64)
		if err != nil {
			return diag.FromErr(err)
		}
		autoScaling := openapi.UpgradeAutoScalingType{}
		if autoScalingState != "" && autoScalingLimitFloat != 0 {
			autoScaling = openapi.UpgradeAutoScalingType{
				State:       &autoScalingState,
				SizeLimitTB: &autoScalingLimitFloat,
			}
		} else if autoScalingState != "" && autoScalingLimitFloat == 0 {
			autoScaling = openapi.UpgradeAutoScalingType{
				State: &autoScalingState,
			}
		} else if autoScalingState == "" && autoScalingLimitFloat != 0 {
			autoScaling = openapi.UpgradeAutoScalingType{
				SizeLimitTB: &autoScalingLimitFloat,
			}
		}

		upgradeObjectStoragaRequest.AutoScaling = &autoScaling
	}

	if anyChange {
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

func StructToMap(obj interface{}) (newMap map[string]interface{}, err error) {
	data, err := json.Marshal(obj) // Convert to a json string
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &newMap) // Convert to a map
	return
}
