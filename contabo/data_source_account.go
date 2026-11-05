package contabo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// contabo_object_storage_credentials

type objectStorageCredential struct {
	AccessKey       string  `json:"accessKey"`
	SecretKey       string  `json:"secretKey"`
	ObjectStorageId string  `json:"objectStorageId"`
	DisplayName     string  `json:"displayName"`
	Region          string  `json:"region"`
	CredentialId    float64 `json:"credentialId"`
}

func dataSourceObjectStorageCredentials() *schema.Resource {
	return &schema.Resource{
		Description: "This data source reads the S3 credentials of the API user. The S3 credentials are an access key and a secret key for an object storage. With this data source, S3 clients and the `contabo_object_storage_bucket` resource do not need keys that you copy by hand. Terraform stores the secret key in the Terraform state.",
		ReadContext: dataSourceObjectStorageCredentialsRead,
		Schema: map[string]*schema.Schema{
			"object_storage_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the data source only returns the credentials of this object storage.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the data source only returns the credentials of object storages in this region, for example `European Union`.",
			},
			"user_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The id of the user whose credentials the data source reads. When you do not set this value, the data source uses the API user of the provider.",
			},
			"access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key of the first credential that matches the filters.",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The secret key of the first credential that matches the filters.",
			},
			"credentials": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "All credentials that match the filters.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"credential_id":     {Type: schema.TypeInt, Computed: true, Description: "The id of the credential."},
						"access_key":        {Type: schema.TypeString, Computed: true, Description: "The S3 access key."},
						"secret_key":        {Type: schema.TypeString, Computed: true, Sensitive: true, Description: "The S3 secret key."},
						"object_storage_id": {Type: schema.TypeString, Computed: true, Description: "The id of the object storage."},
						"display_name":      {Type: schema.TypeString, Computed: true, Description: "The display name of the object storage."},
						"region":            {Type: schema.TypeString, Computed: true, Description: "The region of the object storage."},
					},
				},
			},
		},
	}
}

func dataSourceObjectStorageCredentialsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	userId := d.Get("user_id").(string)
	if userId == "" {
		userId = providerUserId(client)
	}
	if userId == "" {
		return diag.Errorf("could not determine the API user id; set user_id")
	}
	query := url.Values{}
	objectStorageId := d.Get("object_storage_id").(string)
	if objectStorageId != "" {
		query.Set("objectStorageId", objectStorageId)
	}
	region := d.Get("region").(string)
	if region != "" {
		query.Set("regionName", region)
	}

	var credentials []objectStorageCredential
	httpResp, err := listAll(ctx, client, "/v1/users/"+url.PathEscape(userId)+"/object-storages/credentials", query, func(body []byte) (int64, error) {
		var page struct {
			Data       []objectStorageCredential `json:"data"`
			Pagination pagination                `json:"_pagination"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, err
		}
		credentials = append(credentials, page.Data...)
		return page.Pagination.TotalPages, nil
	})
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}

	out := []interface{}{}
	for _, credential := range credentials {
		if objectStorageId != "" && credential.ObjectStorageId != objectStorageId {
			continue
		}
		if region != "" && !strings.EqualFold(credential.Region, region) {
			continue
		}
		out = append(out, map[string]interface{}{
			"credential_id":     int(credential.CredentialId),
			"access_key":        credential.AccessKey,
			"secret_key":        credential.SecretKey,
			"object_storage_id": credential.ObjectStorageId,
			"display_name":      credential.DisplayName,
			"region":            credential.Region,
		})
	}
	if len(out) == 0 {
		return diag.Errorf("no object storage credentials found for user %s", userId)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", userId, objectStorageId, region))
	first := out[0].(map[string]interface{})
	values := map[string]interface{}{
		"user_id":     userId,
		"access_key":  first["access_key"],
		"secret_key":  first["secret_key"],
		"credentials": out,
	}
	return setSchemaValues(d, dataSourceObjectStorageCredentials().Schema, values)
}

// contabo_image_stats

func dataSourceImageStats() *schema.Resource {
	return &schema.Resource{
		Description: "This data source reads how much custom image storage your account uses. Custom image storage is an add-on for the whole account. Check `free_size_mb` before you upload an image with the `contabo_image` resource.",
		ReadContext: dataSourceImageStatsRead,
		Schema: map[string]*schema.Schema{
			"current_images_count": {Type: schema.TypeInt, Computed: true, Description: "The number of custom images in your account."},
			"total_size_mb":        {Type: schema.TypeInt, Computed: true, Description: "The size of the custom image storage in megabytes. The value is 0 when your account has no custom image storage."},
			"used_size_mb":         {Type: schema.TypeInt, Computed: true, Description: "The used part of the custom image storage in megabytes."},
			"free_size_mb":         {Type: schema.TypeInt, Computed: true, Description: "The free part of the custom image storage in megabytes."},
		},
	}
}

func dataSourceImageStatsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	var res struct {
		Data []struct {
			CustomerId         string  `json:"customerId"`
			CurrentImagesCount float64 `json:"currentImagesCount"`
			TotalSizeMb        float64 `json:"totalSizeMb"`
			UsedSizeMb         float64 `json:"usedSizeMb"`
			FreeSizeMb         float64 `json:"freeSizeMb"`
		} `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodGet, "/v1/compute/images/stats", nil, nil, &res)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	if len(res.Data) != 1 {
		return NoDataError(nil)
	}
	stats := res.Data[0]
	d.SetId("image-stats-" + stats.CustomerId)
	return setSchemaValues(d, dataSourceImageStats().Schema, map[string]interface{}{
		"current_images_count": int(stats.CurrentImagesCount),
		"total_size_mb":        int(stats.TotalSizeMb),
		"used_size_mb":         int(stats.UsedSizeMb),
		"free_size_mb":         int(stats.FreeSizeMb),
	})
}

// contabo_instance_available_products

type availableProduct struct {
	ProductId                              string  `json:"productId"`
	Name                                   string  `json:"name"`
	FrontendName                           string  `json:"frontendName"`
	Category                               string  `json:"category"`
	RamSizeGb                              float64 `json:"ramSizeGb"`
	DiskSizeGb                             float64 `json:"diskSizeGb"`
	CpuCores                               float64 `json:"cpuCores"`
	NetSpeed                               float64 `json:"netSpeed"`
	Snapshots                              float64 `json:"snapshots"`
	VsPrice                                float64 `json:"vsPrice"`
	Current                                bool    `json:"current"`
	LocationChangeRequired                 bool    `json:"locationChangeRequired"`
	LiveMigrationAvailable                 bool    `json:"liveMigrationAvailable"`
	LiveMigrationDisabledReason            *string `json:"liveMigrationDisabledReason"`
	StorageExtensionAvailable              bool    `json:"storageExtensionAvailable"`
	BackupAvailable                        bool    `json:"backupAvailable"`
	PriceDifferenceWithStorageExtension    float64 `json:"priceDifferenceWithStorageExtension"`
	PriceDifferenceWithoutStorageExtension float64 `json:"priceDifferenceWithoutStorageExtension"`
}

func dataSourceInstanceAvailableProducts() *schema.Resource {
	computed := func(t schema.ValueType, description string) *schema.Schema {
		return &schema.Schema{Type: t, Computed: true, Description: description}
	}
	return &schema.Resource{
		Description: "This data source lists the products that an existing instance can be upgraded to. You make the upgrade in the Customer Control Panel. This data source shows the options and the prices.",
		ReadContext: dataSourceInstanceAvailableProductsRead,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The id of the instance.",
			},
			"products": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The products that the instance can be upgraded to.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"product_id":                     computed(schema.TypeString, "The product id, for example `V154`."),
						"name":                           computed(schema.TypeString, "The name of the product."),
						"frontend_name":                  computed(schema.TypeString, "The name of the product that the Customer Control Panel shows."),
						"category":                       computed(schema.TypeString, "The category of the product, for example `ssd` or `nvme`."),
						"ram_size_gb":                    computed(schema.TypeInt, "The size of the memory (RAM) in gigabytes."),
						"disk_size_gb":                   computed(schema.TypeInt, "The size of the disk in gigabytes."),
						"cpu_cores":                      computed(schema.TypeInt, "The number of CPU cores."),
						"net_speed":                      computed(schema.TypeInt, "The network speed in Mbit/s."),
						"snapshots":                      computed(schema.TypeInt, "The number of snapshots that the product includes."),
						"monthly_price":                  computed(schema.TypeFloat, "The monthly price of the product."),
						"current":                        computed(schema.TypeBool, "Whether this product is the current product of the instance."),
						"location_change_required":       computed(schema.TypeBool, "Whether the upgrade moves the instance to another location."),
						"live_migration_available":       computed(schema.TypeBool, "Whether the upgrade can run without downtime."),
						"live_migration_disabled_reason": computed(schema.TypeString, "The reason why the upgrade cannot run without downtime."),
						"storage_extension_available":    computed(schema.TypeBool, "Whether you can add a storage extension to the product."),
						"backup_available":               computed(schema.TypeBool, "Whether automated backups are available for the product."),
						"price_difference":               computed(schema.TypeFloat, "The price difference of the upgrade without a storage extension."),
						"price_difference_with_storage":  computed(schema.TypeFloat, "The price difference of the upgrade with a storage extension."),
					},
				},
			},
		},
	}
}

func dataSourceInstanceAvailableProductsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Get("instance_id").(string), 10, 64)
	if err != nil {
		return diag.Errorf("instance_id must be a numeric instance id: %s", err)
	}
	var res struct {
		Data []availableProduct `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodGet, instancePath(instanceId)+"/products/available", nil, nil, &res)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	products := make([]interface{}, 0, len(res.Data))
	for _, product := range res.Data {
		reason := ""
		if product.LiveMigrationDisabledReason != nil {
			reason = *product.LiveMigrationDisabledReason
		}
		products = append(products, map[string]interface{}{
			"product_id":                     product.ProductId,
			"name":                           product.Name,
			"frontend_name":                  product.FrontendName,
			"category":                       product.Category,
			"ram_size_gb":                    int(product.RamSizeGb),
			"disk_size_gb":                   int(product.DiskSizeGb),
			"cpu_cores":                      int(product.CpuCores),
			"net_speed":                      int(product.NetSpeed),
			"snapshots":                      int(product.Snapshots),
			"monthly_price":                  product.VsPrice,
			"current":                        product.Current,
			"location_change_required":       product.LocationChangeRequired,
			"live_migration_available":       product.LiveMigrationAvailable,
			"live_migration_disabled_reason": reason,
			"storage_extension_available":    product.StorageExtensionAvailable,
			"backup_available":               product.BackupAvailable,
			"price_difference":               product.PriceDifferenceWithoutStorageExtension,
			"price_difference_with_storage":  product.PriceDifferenceWithStorageExtension,
		})
	}
	d.SetId(strconv.FormatInt(instanceId, 10))
	if err := d.Set("products", products); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// contabo_instance_vnc

func dataSourceInstanceVnc() *schema.Resource {
	return &schema.Resource{
		Description: "This data source reads the VNC console settings of an instance.",
		ReadContext: dataSourceInstanceVncRead,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The id of the instance.",
			},
			"enabled":  {Type: schema.TypeBool, Computed: true, Description: "Whether the VNC console of the instance is enabled."},
			"vnc_ip":   {Type: schema.TypeString, Computed: true, Description: "The IP address of the VNC console."},
			"vnc_port": {Type: schema.TypeInt, Computed: true, Description: "The port of the VNC console."},
		},
	}
}

type vncResponse struct {
	InstanceId int64   `json:"instanceId"`
	Enabled    bool    `json:"enabled"`
	VncIp      string  `json:"vncIp"`
	VncPort    float64 `json:"vncPort"`
}

func retrieveInstanceVnc(ctx context.Context, client *openapi.APIClient, instanceId int64) (*vncResponse, *http.Response, error) {
	var res struct {
		Data []vncResponse `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodGet, instancePath(instanceId)+"/vnc", nil, nil, &res)
	if err != nil {
		return nil, httpResp, err
	}
	if len(res.Data) != 1 {
		return nil, httpResp, fmt.Errorf("expected VNC settings of instance %d", instanceId)
	}
	return &res.Data[0], httpResp, nil
}

func dataSourceInstanceVncRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Get("instance_id").(string), 10, 64)
	if err != nil {
		return diag.Errorf("instance_id must be a numeric instance id: %s", err)
	}
	vnc, httpResp, err := retrieveInstanceVnc(ctx, client, instanceId)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	d.SetId(strconv.FormatInt(instanceId, 10))
	return setSchemaValues(d, dataSourceInstanceVnc().Schema, map[string]interface{}{
		"enabled":  vnc.Enabled,
		"vnc_ip":   vnc.VncIp,
		"vnc_port": int(vnc.VncPort),
	})
}
