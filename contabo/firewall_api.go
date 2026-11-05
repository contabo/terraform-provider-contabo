package contabo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"contabo.com/openapi"
)

// firewallResponse mirrors the firewall object of the API. It is decoded from
// the raw response because the API does not match the generated model: it
// returns "instances" as a JSON-encoded string such as "[203235786]" instead
// of an array of objects, which makes the generated client fail to decode
// every firewall that has instances.
type firewallResponse struct {
	FirewallId     string                   `json:"firewallId"`
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	Status         string                   `json:"status"`
	InstanceStatus []firewallInstanceStatus `json:"instanceStatus"`
	Instances      json.RawMessage          `json:"instances"`
	Rules          struct {
		Inbound []firewallRule `json:"inbound"`
	} `json:"rules"`
	CreatedDate time.Time `json:"createdDate"`
}

type firewallInstanceStatus struct {
	InstanceId   int64   `json:"instanceId"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"errorMessage"`
}

type firewallRule struct {
	Protocol    string   `json:"protocol"`
	DestPorts   []string `json:"destPorts"`
	SrcCidr     srcCidr  `json:"srcCidr"`
	Action      string   `json:"action"`
	Status      string   `json:"status"`
	DisplayName string   `json:"displayName,omitempty"`
}

type srcCidr struct {
	Ipv4 []string `json:"ipv4,omitempty"`
	Ipv6 []string `json:"ipv6,omitempty"`
}

type firewallListResponse struct {
	Data       []firewallResponse `json:"data"`
	Pagination pagination         `json:"_pagination"`
}

// instanceIds returns the ids of the instances assigned to the firewall.
// instanceStatus is the reliable source; the "instances" attribute is parsed
// as a fallback in all the shapes the API has used.
func (f firewallResponse) instanceIds() []int64 {
	seen := map[int64]bool{}
	var ids []int64
	add := func(id int64) {
		if id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, status := range f.InstanceStatus {
		add(status.InstanceId)
	}
	for _, id := range parseFirewallInstances(f.Instances) {
		add(id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// parseFirewallInstances accepts "[1,2]" (a JSON-encoded string), [1,2] and
// [{"instanceId":1}].
func parseFirewallInstances(raw json.RawMessage) []int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		encoded = strings.TrimSpace(encoded)
		if encoded == "" {
			return nil
		}
		raw = json.RawMessage(encoded)
	}

	var numbers []json.Number
	if err := json.Unmarshal(raw, &numbers); err == nil {
		ids := make([]int64, 0, len(numbers))
		for _, number := range numbers {
			if id, err := strconv.ParseInt(number.String(), 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		return ids
	}

	var objects []struct {
		InstanceId int64 `json:"instanceId"`
	}
	if err := json.Unmarshal(raw, &objects); err == nil {
		ids := make([]int64, 0, len(objects))
		for _, object := range objects {
			ids = append(ids, object.InstanceId)
		}
		return ids
	}
	return nil
}

func retrieveFirewall(
	ctx context.Context,
	client *openapi.APIClient,
	firewallId string,
) (*firewallResponse, *http.Response, error) {
	var res struct {
		Data []firewallResponse `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodGet, "/v1/firewalls/"+url.PathEscape(firewallId), nil, nil, &res)
	if err != nil {
		return nil, httpResp, err
	}
	if len(res.Data) != 1 {
		return nil, httpResp, fmt.Errorf("expected one firewall with id %s, got %d", firewallId, len(res.Data))
	}
	return &res.Data[0], httpResp, nil
}

func listFirewalls(
	ctx context.Context,
	client *openapi.APIClient,
	query url.Values,
) ([]firewallResponse, *http.Response, error) {
	var firewalls []firewallResponse
	httpResp, err := listAll(ctx, client, "/v1/firewalls", query, func(body []byte) (int64, error) {
		var page firewallListResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, err
		}
		firewalls = append(firewalls, page.Data...)
		return page.Pagination.TotalPages, nil
	})
	return firewalls, httpResp, err
}
