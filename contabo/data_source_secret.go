package contabo

import (
	"context"
	"strconv"

	apiClient "contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourceSecret() *schema.Resource {
	return &schema.Resource{
		Description: "The Secret Management API allows you to store and manage your passwords and ssh-keys. Usage of the Secret Management API is purely optional. As a convenience feature e.g. it allows you to reuse SSH-keys easily.",
		ReadContext: dataSourceSecretRead,
		Schema: map[string]*schema.Schema{
			"created_at": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The creation date of the secret.",
			},
			"updated_at": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Last updated time of the secret.",
			},
			"id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the secret. Use it to manage it!",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the secret.",
			},
			"value": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The value of the secret. It will be available only when retrieving a single secret.",
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The type of the secret. It will be available only when retrieving secrets, following types are allowed: `ssh`, `password`.",
			},
		},
	}
}

func dataSourceSecretRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*apiClient.APIClient)

	var secretId int64
	var err error
	id := d.Get("id").(string)
	if id != "" {
		secretId, err = strconv.ParseInt(id, 10, 64)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.SecretsApi.
		RetrieveSecret(ctx, secretId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(strconv.Itoa(int(res.Data[0].SecretId)))

	return AddSecretToData(res.Data[0], d, diags)
}
