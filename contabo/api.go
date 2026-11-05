package contabo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

// newRequestId returns a value for the mandatory x-request-id header.
func newRequestId() string {
	return uuid.NewV4().String()
}

// isNotFound is defined in dns_utils.go.

// apiRequest sends a request with the provider's authenticated HTTP client
// and decodes the JSON response into out (when out is not nil). It covers
// endpoints that the generated client does not offer or models wrongly.
//
// A response with a status code of 300 or more returns an error together
// with the response, so callers can check isNotFound or build diagnostics
// with apiErrorDiags.
func apiRequest(
	ctx context.Context,
	client *openapi.APIClient,
	method string,
	path string,
	query url.Values,
	body interface{},
	out interface{},
) (*http.Response, error) {
	config := client.GetConfig()
	if len(config.Servers) == 0 {
		return nil, fmt.Errorf("no API server configured")
	}
	endpoint := strings.TrimRight(config.Servers[0].URL, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("x-request-id", newRequestId())
	for key, value := range config.DefaultHeader {
		req.Header.Set(key, value)
	}
	if config.UserAgent != "" {
		req.Header.Set("User-Agent", config.UserAgent)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpResp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(httpResp.Body)
	_ = httpResp.Body.Close()
	httpResp.Body = io.NopCloser(bytes.NewReader(respBody))
	if err != nil {
		return httpResp, err
	}
	if httpResp.StatusCode >= 300 {
		return httpResp, fmt.Errorf("%s %s: %s", method, path, httpResp.Status)
	}
	if out != nil && len(bytes.TrimSpace(respBody)) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return httpResp, fmt.Errorf("could not decode API response: %w", err)
		}
	}
	return httpResp, nil
}

// pagination is the _pagination object of list responses.
type pagination struct {
	Size          int64 `json:"size"`
	TotalElements int64 `json:"totalElements"`
	TotalPages    int64 `json:"totalPages"`
	Page          int64 `json:"page"`
}

// listAll fetches every page of a list endpoint. decodePage receives each
// raw page and returns the number of total pages.
func listAll(
	ctx context.Context,
	client *openapi.APIClient,
	path string,
	query url.Values,
	decodePage func(body []byte) (int64, error),
) (*http.Response, error) {
	const pageSize = 100
	for page := int64(1); ; page++ {
		pageQuery := url.Values{}
		for key, values := range query {
			pageQuery[key] = values
		}
		pageQuery.Set("page", fmt.Sprint(page))
		pageQuery.Set("size", fmt.Sprint(pageSize))

		var raw json.RawMessage
		httpResp, err := apiRequest(ctx, client, http.MethodGet, path, pageQuery, nil, &raw)
		if err != nil {
			return httpResp, err
		}
		totalPages, err := decodePage(raw)
		if err != nil {
			return httpResp, err
		}
		if page >= totalPages {
			return httpResp, nil
		}
	}
}

// waitFor polls check until it reports done, returns an error, or the
// timeout passes.
func waitFor(
	ctx context.Context,
	timeout time.Duration,
	interval time.Duration,
	check func() (done bool, err error),
) error {
	deadline := time.Now().Add(timeout)
	for {
		done, err := check()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// pollInterval is shortened by the unit tests.
var pollInterval = 5 * time.Second

// warningDiag returns a warning diagnostic.
func warningDiag(summary string, detail string) diag.Diagnostic {
	return diag.Diagnostic{Severity: diag.Warning, Summary: summary, Detail: detail}
}

// removeFromStateIfNotFound clears the resource id when the API reports the
// object as gone, so that Terraform plans to create it again instead of
// failing. It returns true when the caller should stop reading.
func removeFromStateIfNotFound(d *schema.ResourceData, httpResp *http.Response) bool {
	if isNotFound(httpResp) {
		d.SetId("")
		return true
	}
	return false
}

func stringSetToSlice(set *schema.Set) []string {
	if set == nil {
		return nil
	}
	list := set.List()
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func jsonUnmarshal(body []byte, out interface{}) error {
	return json.Unmarshal(body, out)
}

// hashStrings returns a stable hash, used for data source ids.
func hashStrings(values []string) uint64 {
	var hash uint64 = 14695981039346656037
	for _, value := range values {
		for i := 0; i < len(value); i++ {
			hash ^= uint64(value[i])
			hash *= 1099511628211
		}
		hash ^= 0xff
		hash *= 1099511628211
	}
	return hash
}
