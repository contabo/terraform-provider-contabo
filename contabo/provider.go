package contabo

import (
	"context"
	"net/url"

	"contabo.com/terraform-provider-contabo/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_API", "https://api.contabo.com"),
			},
			"oauth2_token_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_TOKEN_URL", "https://auth.contabo.com/auth/realms/contabo/protocol/openid-connect/token"),
			},
			"oauth2_client_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_CLIENT_ID", nil),
			},
			"oauth2_client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_CLIENT_SECRET", nil),
			},
			"oauth2_user": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_USER", nil),
			},
			"oauth2_pass": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_PASS", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"contabo_instance":          resourceInstance(),
			"contabo_instance_snapshot": resourceSnapshot(),
			"contabo_image":             resourceImage(),
			"contabo_object_storage":    resourceObjectStorage(),
			"contabo_secret":            resourceSecret(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"contabo_instance":          dataSourceInstance(),
			"contabo_instance_snapshot": dataSourceSnapshot(),
			"contabo_image":             dataSourceImage(),
			"contabo_object_storage":    dataSourceObjectStorage(),
			"contabo_secret":            dataSourceSecret(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(
	ctx context.Context,
	d *schema.ResourceData,
) (interface{}, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics
	apiUrl := d.Get("api").(string)
	authUrl := d.Get("oauth2_token_url").(string)
	clientId := d.Get("oauth2_client_id").(string)
	clientSecret := d.Get("oauth2_client_secret").(string)
	username := d.Get("oauth2_user").(string)
	password := d.Get("oauth2_pass").(string)

	parsedTokenUrl, err := url.ParseRequestURI(authUrl)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	// TODO: validate config values
	newClient, err := client.NewClient(
		apiUrl,
		parsedTokenUrl.String(),
		clientId,
		&clientSecret,
		username,
		&password,
	)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return newClient, diags
}
