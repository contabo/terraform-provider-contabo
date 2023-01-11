package contabo

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"contabo.com/terraform-provider-contabo/client"
	"github.com/golang-jwt/jwt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var userId string

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_API", "https://api.contabo.com"),
				Description: "The api endpoint is https://api.contabo.com.",
			},
			"oauth2_token_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_TOKEN_URL", "https://auth.contabo.com/auth/realms/contabo/protocol/openid-connect/token"),
				Description: "The oauth2 token url is https://auth.contabo.com/auth/realms/contabo/protocol/openid-connect/token.",
			},
			"oauth2_client_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_CLIENT_ID", nil),
				Description: "Your oauth2 client id can be found in the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.",
			},
			"oauth2_client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_CLIENT_SECRET", nil),
				Description: "Your oauth2 client secret can be found in the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.",
			},
			"oauth2_user": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CNTB_OAUTH2_USER", nil),
				Description: "API User (your email address to login to the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.",
			},
			"oauth2_pass": &schema.Schema{
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
			"contabo_private_network":       resourcePrivateNetwork(),
			"contabo_object_storage_bucket": resourceObjectStorageBucket(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"contabo_instance":              dataSourceInstance(),
			"contabo_instance_snapshot":     dataSourceSnapshot(),
			"contabo_image":                 dataSourceImage(),
			"contabo_object_storage":        dataSourceObjectStorage(),
			"contabo_secret":                dataSourceSecret(),
			"contabo_private_network":       dataSourcePrivateNetwork(),
			"contabo_object_storage_bucket": dataSourceObjectStorageBucket(),
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

	userId, diags = getUserId(
		diags,
		parsedTokenUrl.String(),
		clientId,
		clientSecret,
		username,
		password,
	)
	return newClient, diags
}

func getUserId(
	diags diag.Diagnostics,
	authUrl string,
	clientId string,
	clientSecret string,
	username string,
	password string,
) (string, diag.Diagnostics) {

	jwtToken, diags := GetJwtToken(diags, authUrl, clientId, clientSecret, username, password)

	if (JwtToken{}) == jwtToken {
		return "", diag.FromErr(errors.New("error in getting jwt token"))
	}

	claims := jwt.MapClaims{}

	_, err := jwt.ParseWithClaims(strings.TrimSpace(jwtToken.AccessToken), claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("<YOUR VERIFICATION KEY>"), nil
	})

	if err != nil {
		//TODO This throws an error but in the cntb we just ignoring it... :)
		// return "", diag.FromErr(err)
	}

	if claims["sub"] == nil {
		return "", diag.FromErr(errors.New("error in getting access token"))
	}
	return claims["sub"].(string), diags
}

type JwtToken struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int64  `json:"expires_in"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int64  `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

func GetJwtToken(
	diags diag.Diagnostics,
	authUrl string,
	clientId string,
	clientSecret string,
	username string,
	password string,
) (JwtToken, diag.Diagnostics) {
	var jwtToken JwtToken

	urlEncodedUsername := url.QueryEscape(username)
	urlEncodedPassword := url.QueryEscape(password)

	payload := strings.NewReader("client_id=" + clientId + "&client_secret=" + clientSecret + "&username=" + urlEncodedUsername + "&password=" + urlEncodedPassword + "&grant_type=password")

	client := &http.Client{}
	req, err := http.NewRequest("POST", authUrl, payload)

	if err != nil {
		return jwtToken, diag.FromErr(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return jwtToken, diag.FromErr(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return jwtToken, diag.FromErr(err)
	}

	if err := json.Unmarshal(body, &jwtToken); err != nil {
		return jwtToken, diag.FromErr(err)
	}

	return jwtToken, diags
}
