package contabo

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"sync"

	"contabo.com/openapi"
	"contabo.com/terraform-provider-contabo/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_API", "https://api.contabo.com"),
				Description: "The api endpoint is https://api.contabo.com.",
			},
			"oauth2_token_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_TOKEN_URL", "https://auth.contabo.com/auth/realms/contabo/protocol/openid-connect/token"),
				Description: "The oauth2 token url is https://auth.contabo.com/auth/realms/contabo/protocol/openid-connect/token.",
			},
			"oauth2_client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_CLIENT_ID", nil),
				Description: "Your oauth2 client id can be found in the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.",
			},
			"oauth2_client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_CLIENT_SECRET", nil),
				Description: "Your oauth2 client secret can be found in the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.",
			},
			"oauth2_user": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_USER", nil),
				Description: "API User (your email address to login to the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.",
			},
			"oauth2_pass": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_PASS", nil),
				Description: "API Password (this is a new password which you'll set or change in the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.)",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"contabo_instance":              resourceInstance(),
			"contabo_instance_snapshot":     resourceSnapshot(),
			"contabo_image":                 resourceImage(),
			"contabo_object_storage":        resourceObjectStorage(),
			"contabo_secret":                resourceSecret(),
			"contabo_firewall":              resourceFirewall(),
			"contabo_private_network":       resourcePrivateNetwork(),
			"contabo_object_storage_bucket": resourceObjectStorageBucket(),
			"contabo_tag":                   resourceTag(),
			"contabo_tag_assignment":        resourceTagAssignment(),
			"contabo_dns_zone":              resourceDnsZone(),
			"contabo_dns_record":            resourceDnsRecord(),
			"contabo_ptr_record":            resourcePtrRecord(),
			"contabo_instance_reinstall":    resourceInstanceReinstall(),
			"contabo_instance_vnc":          resourceInstanceVnc(),
			"contabo_vip_assignment":        resourceVipAssignment(),
			"contabo_role":                  resourceRole(),
			"contabo_user":                  resourceUser(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"contabo_instance":                    dataSourceInstance(),
			"contabo_instance_snapshot":           dataSourceSnapshot(),
			"contabo_image":                       dataSourceImage(),
			"contabo_object_storage":              dataSourceObjectStorage(),
			"contabo_secret":                      dataSourceSecret(),
			"contabo_firewall":                    dataSourceFirewall(),
			"contabo_private_network":             dataSourcePrivateNetwork(),
			"contabo_object_storage_bucket":       dataSourceObjectStorageBucket(),
			"contabo_tag":                         dataSourceTag(),
			"contabo_tag_assignment":              dataSourceTagAssignment(),
			"contabo_dns_zone":                    dataSourceDnsZone(),
			"contabo_ptr_record":                  dataSourcePtrRecord(),
			"contabo_instances":                   dataSourceInstances(),
			"contabo_vip":                         dataSourceVip(),
			"contabo_vips":                        dataSourceVips(),
			"contabo_object_storage_credentials":  dataSourceObjectStorageCredentials(),
			"contabo_image_stats":                 dataSourceImageStats(),
			"contabo_instance_available_products": dataSourceInstanceAvailableProducts(),
			"contabo_instance_vnc":                dataSourceInstanceVnc(),
			"contabo_role":                        dataSourceRole(),
			"contabo_api_permissions":             dataSourceApiPermissions(),
			"contabo_users":                       dataSourceUsers(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(
	ctx context.Context,
	d *schema.ResourceData,
) (interface{}, diag.Diagnostics) {
	apiUrl := d.Get("api").(string)
	authUrl := d.Get("oauth2_token_url").(string)
	clientId := d.Get("oauth2_client_id").(string)
	clientSecret := d.Get("oauth2_client_secret").(string)
	username := d.Get("oauth2_user").(string)
	password := d.Get("oauth2_pass").(string)

	var missing []string
	for name, value := range map[string]string{
		"oauth2_client_id":     clientId,
		"oauth2_client_secret": clientSecret,
		"oauth2_user":          username,
		"oauth2_pass":          password,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, diag.Errorf(
			"missing provider credentials: %s. Set them in the provider block or with the CNTB_OAUTH2_* environment variables",
			strings.Join(missing, ", "),
		)
	}

	parsedTokenUrl, err := url.ParseRequestURI(authUrl)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	httpClient, err := client.BearerHttpClient(
		parsedTokenUrl.String(),
		clientId,
		clientSecret,
		username,
		password,
	)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	userId, err := client.AccessTokenSubject(httpClient)
	if err != nil {
		return nil, diag.Errorf("could not determine the API user id: %s", err)
	}

	apiClient := client.NewClientWithHTTPClient(apiUrl, httpClient)
	setProviderUserId(apiClient, userId)
	return apiClient, nil
}

// providerUserIds maps each configured API client to the Contabo user id of
// its API user. Provider aliases may use different accounts, so the id
// cannot be a single package variable.
var providerUserIds sync.Map

func setProviderUserId(apiClient *openapi.APIClient, userId string) {
	providerUserIds.Store(apiClient, userId)
}

func providerUserId(apiClient *openapi.APIClient) string {
	userId, _ := providerUserIds.Load(apiClient)
	s, _ := userId.(string)
	return s
}
