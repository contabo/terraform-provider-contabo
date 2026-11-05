package client

import (
	"net/http"

	"contabo.com/openapi"
)

// UserAgent is sent with every API request. main sets the version.
var UserAgent = "terraform-provider-contabo"

func NewClient(
	apiUrl string,
	authUrl string,
	clientId string,
	clientSecret *string,
	username string,
	password *string,
) (*openapi.APIClient, error) {
	httpClient, err := BearerHttpClient(
		authUrl,
		clientId,
		*clientSecret,
		username,
		*password,
	)
	if err != nil {
		return nil, err
	}
	return NewClientWithHTTPClient(apiUrl, httpClient), nil
}

// NewClientWithHTTPClient returns an API client that sends requests to apiUrl
// with the given HTTP client, which must add authentication.
func NewClientWithHTTPClient(apiUrl string, httpClient *http.Client) *openapi.APIClient {
	configuration := openapi.NewConfiguration()
	configuration.AddDefaultHeader("x-trace-id", "contabo_terraform_provider")
	configuration.UserAgent = UserAgent
	configuration.HTTPClient = httpClient
	configuration.Servers = openapi.ServerConfigurations{{URL: apiUrl}}
	return openapi.NewAPIClient(configuration)
}
