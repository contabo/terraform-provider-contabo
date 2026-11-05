package contabo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// instanceResponse mirrors the instance object of the API. It is decoded from
// the raw response instead of the generated model, because the generated
// model rejects status values that are not in its enum, and the API adds
// statuses over time.
type instanceResponse struct {
	TenantId      string              `json:"tenantId"`
	CustomerId    string              `json:"customerId"`
	AdditionalIps []instanceIpWrapper `json:"additionalIps"`
	Name          string              `json:"name"`
	DisplayName   string              `json:"displayName"`
	InstanceId    int64               `json:"instanceId"`
	DataCenter    string              `json:"dataCenter"`
	Region        string              `json:"region"`
	RegionName    string              `json:"regionName"`
	ProductId     string              `json:"productId"`
	ImageId       *string             `json:"imageId"`
	IpConfig      struct {
		V4 instanceIp `json:"v4"`
		V6 instanceIp `json:"v6"`
	} `json:"ipConfig"`
	MacAddress    string          `json:"macAddress"`
	RamMb         float64         `json:"ramMb"`
	CpuCores      int64           `json:"cpuCores"`
	OsType        string          `json:"osType"`
	DiskMb        float64         `json:"diskMb"`
	SshKeys       []int64         `json:"sshKeys"`
	CreatedDate   time.Time       `json:"createdDate"`
	CancelDate    *string         `json:"cancelDate"`
	Status        string          `json:"status"`
	VHostId       int64           `json:"vHostId"`
	VHostNumber   int64           `json:"vHostNumber"`
	VHostName     string          `json:"vHostName"`
	AddOns        []instanceAddOn `json:"addOns"`
	ErrorMessage  *string         `json:"errorMessage"`
	ProductType   string          `json:"productType"`
	ProductName   string          `json:"productName"`
	DefaultUser   *string         `json:"defaultUser"`
	ApplicationId *string         `json:"applicationId"`
	GpuVendor     *string         `json:"gpuVendor"`
	GpuModel      *string         `json:"gpuModel"`
}

type instanceIp struct {
	Ip          string  `json:"ip"`
	NetmaskCidr float64 `json:"netmaskCidr"`
	Gateway     string  `json:"gateway"`
}

type instanceIpWrapper struct {
	V4 instanceIp `json:"v4"`
}

type instanceAddOn struct {
	Id       int64 `json:"id"`
	Quantity int64 `json:"quantity"`
}

type instanceListResponse struct {
	Data       []instanceResponse `json:"data"`
	Pagination pagination         `json:"_pagination"`
}

func (i instanceResponse) cancelDate() string {
	if i.CancelDate == nil {
		return ""
	}
	return *i.CancelDate
}

func (i instanceResponse) errorMessage() string {
	if i.ErrorMessage == nil {
		return ""
	}
	return *i.ErrorMessage
}

// Instance statuses during which the instance is still being set up.
var instanceTransitionalStatuses = map[string]bool{
	"provisioning":   true,
	"installing":     true,
	"reset_password": true,
	"unknown":        true,
}

// Instance statuses that need action outside the API (payment, identity
// verification, a support ticket). Waiting for them would block until the
// timeout, so the provider stops waiting and reports a warning.
var instanceBlockedStatuses = map[string]bool{
	"manual_provisioning":   true,
	"product_not_available": true,
	"verification_required": true,
	"pending_payment":       true,
}

func instancePath(instanceId int64) string {
	return fmt.Sprintf("/v1/compute/instances/%d", instanceId)
}

func retrieveInstance(
	ctx context.Context,
	client *openapi.APIClient,
	instanceId int64,
) (*instanceResponse, *http.Response, error) {
	var res instanceListResponse
	httpResp, err := apiRequest(ctx, client, http.MethodGet, instancePath(instanceId), nil, nil, &res)
	if err != nil {
		return nil, httpResp, err
	}
	if len(res.Data) != 1 {
		return nil, httpResp, fmt.Errorf("expected one instance with id %d, got %d", instanceId, len(res.Data))
	}
	return &res.Data[0], httpResp, nil
}

func listInstances(
	ctx context.Context,
	client *openapi.APIClient,
	query url.Values,
) ([]instanceResponse, *http.Response, error) {
	var instances []instanceResponse
	httpResp, err := listAll(ctx, client, "/v1/compute/instances", query, func(body []byte) (int64, error) {
		var page instanceListResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, err
		}
		instances = append(instances, page.Data...)
		return page.Pagination.TotalPages, nil
	})
	return instances, httpResp, err
}

// waitForInstanceReady waits until the instance has left the provisioning and
// installation states. It fails when the instance ends in the error state and
// warns when the instance waits for something the API cannot do (payment,
// verification, manual provisioning).
func waitForInstanceReady(
	ctx context.Context,
	client *openapi.APIClient,
	instanceId int64,
	timeout time.Duration,
) (*instanceResponse, diag.Diagnostics) {
	var instance *instanceResponse
	var diags diag.Diagnostics
	err := waitFor(ctx, timeout, pollInterval, func() (bool, error) {
		current, httpResp, err := retrieveInstance(ctx, client, instanceId)
		if err != nil {
			// The instance may not be visible for a moment after creation.
			if isNotFound(httpResp) {
				return false, nil
			}
			diags = apiErrorDiags(httpResp, err)
			return false, err
		}
		instance = current
		status := strings.ToLower(current.Status)
		switch {
		case status == "error":
			diags = diag.Errorf("instance %d is in the error state: %s", instanceId, current.errorMessage())
			return false, fmt.Errorf("instance error")
		case instanceBlockedStatuses[status]:
			diags = append(diags, warningDiag(
				fmt.Sprintf("Instance %d is %s", instanceId, status),
				"The instance waits for an action outside the API, for example a payment or an identity verification in the customer control panel. The provider stopped waiting for it.",
			))
			return true, nil
		case instanceTransitionalStatuses[status]:
			return false, nil
		default:
			return true, nil
		}
	})
	if err != nil {
		if diags.HasError() {
			return instance, diags
		}
		return instance, diag.Errorf("waiting for instance %d to be ready: %s", instanceId, err)
	}
	return instance, diags
}
