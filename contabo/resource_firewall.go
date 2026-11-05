package contabo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const inboundRuleKey = "inbound"

// firewallProtocolAny is how the configuration spells a rule for every
// protocol. The API represents it as an empty protocol.
const firewallProtocolAny = "any"

func firewallRuleSchema(computedOnly bool) *schema.Resource {
	optional := !computedOnly
	var suppressProtocol schema.SchemaDiffSuppressFunc
	if optional {
		suppressProtocol = suppressFirewallProtocolDiff
	}
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"protocol": {
				Type:             schema.TypeString,
				Computed:         true,
				Optional:         optional,
				DiffSuppressFunc: suppressProtocol,
				Description:      "The protocol of the rule. The value is `tcp`, `udp`, `icmp`, or `any` for every protocol.",
			},
			"action": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    optional,
				Description: "The action of the rule, for example `accept` or `drop`. The default value is `accept`.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    optional,
				Description: "The status of the rule. The value is `active` or `inactive`. The default value is `active`.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    optional,
				Description: "The name of the rule. The Customer Control Panel shows this name.",
			},
			"dest_ports": {
				Type:        schema.TypeSet,
				Computed:    computedOnly,
				Optional:    optional,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The destination ports or port ranges of the rule, for example `22` or `8000-8080`. When the list is empty, the rule applies to every port.",
			},
			"src_cidr": {
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    optional,
				Description: "The source addresses that the rule applies to.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ipv4": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Computed:    computedOnly,
							Optional:    optional,
							Description: "The IPv4 addresses or CIDR ranges, for example `0.0.0.0/0`.",
						},
						"ipv6": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Computed:    computedOnly,
							Optional:    optional,
							Description: "The IPv6 addresses or CIDR ranges, for example `::/0`.",
						},
					},
				},
			},
		},
	}
}

func firewallInstancesStatusSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The status of the firewall on every instance that the firewall protects.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"instance_id": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The id of the instance.",
				},
				"status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The status of the firewall on the instance.",
				},
				"error_message": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The error message when the firewall could not be applied to the instance.",
				},
			},
		},
	}
}

func resourceFirewall() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource manages a firewall and the instances that the firewall protects. Firewalls are free, and a firewall works on every instance. The API adds a rule to the end of every firewall that blocks all other inbound traffic. The provider does not show that rule.",
		CreateContext: resourceFirewallCreate,
		UpdateContext: resourceFirewallUpdate,
		ReadContext:   resourceFirewallRead,
		DeleteContext: resourceFirewallDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"created_date": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The creation date of the Firewall.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the Firewall. Use it to manage it!",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
				Description:  "The name of the Firewall. It may contain letters, numbers, colons, dashes, and underscores. There is a limit of 255 characters per Firewall name.",
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 255),
				Description:  "The description of the Firewall. There is a limit of 255 characters per Firewall.",
			},
			"status": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"active", "inactive"}, false),
				Description:  "Status of the Firewall. It can be `active`, or `inactive`.",
			},
			"instance_ids": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Optional:    true,
				Description: "The ids of the instances that the firewall protects. Leave the list empty for a firewall without instances. Before the provider adds an instance to the firewall, the provider waits until the instance is running.",
			},
			"instances_status": firewallInstancesStatusSchema(),
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				MaxItems:    1,
				Description: "The rules of the firewall.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"inbound": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The inbound rules of the firewall. The firewall drops all inbound traffic that no active rule accepts.",
							Elem:        firewallRuleSchema(false),
						},
					},
				},
			},
		},
	}
}

func suppressFirewallProtocolDiff(_, old, new string, _ *schema.ResourceData) bool {
	return normalizeFirewallProtocol(old) == normalizeFirewallProtocol(new)
}

func normalizeFirewallProtocol(protocol string) string {
	if strings.EqualFold(protocol, firewallProtocolAny) {
		return ""
	}
	return strings.ToLower(protocol)
}

// expandFirewallRules builds the API rules from the configuration. Empty
// nested blocks arrive as nil and are treated as absent.
func expandFirewallRules(d *schema.ResourceData) []firewallRule {
	rules := []firewallRule{}
	rulesList, _ := d.Get("rules").([]interface{})
	if len(rulesList) == 0 || rulesList[0] == nil {
		return rules
	}
	rulesMap, ok := rulesList[0].(map[string]interface{})
	if !ok {
		return rules
	}
	inbound, _ := rulesMap[inboundRuleKey].([]interface{})
	for _, item := range inbound {
		ruleMap, ok := item.(map[string]interface{})
		if !ok || ruleMap == nil {
			continue
		}
		rule := firewallRule{
			Protocol:    normalizeFirewallProtocol(mapString(ruleMap, "protocol")),
			Action:      mapString(ruleMap, "action"),
			Status:      mapString(ruleMap, "status"),
			DisplayName: mapString(ruleMap, "display_name"),
			DestPorts:   []string{},
		}
		if rule.Action == "" {
			rule.Action = "accept"
		}
		if rule.Status == "" {
			rule.Status = "active"
		}
		if ports, ok := ruleMap["dest_ports"].(*schema.Set); ok {
			rule.DestPorts = stringSetToSlice(ports)
			sort.Strings(rule.DestPorts)
		}
		if cidrs, ok := ruleMap["src_cidr"].([]interface{}); ok {
			for _, cidr := range cidrs {
				cidrMap, ok := cidr.(map[string]interface{})
				if !ok || cidrMap == nil {
					continue
				}
				if ipv4, ok := cidrMap["ipv4"].(*schema.Set); ok {
					rule.SrcCidr.Ipv4 = append(rule.SrcCidr.Ipv4, stringSetToSlice(ipv4)...)
				}
				if ipv6, ok := cidrMap["ipv6"].(*schema.Set); ok {
					rule.SrcCidr.Ipv6 = append(rule.SrcCidr.Ipv6, stringSetToSlice(ipv6)...)
				}
			}
		}
		if rule.SrcCidr.Ipv4 == nil {
			rule.SrcCidr.Ipv4 = []string{}
		}
		if rule.SrcCidr.Ipv6 == nil {
			rule.SrcCidr.Ipv6 = []string{}
		}
		rules = append(rules, rule)
	}
	return rules
}

func mapString(m map[string]interface{}, key string) string {
	s, _ := m[key].(string)
	return s
}

func resourceFirewallCreate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	client := m.(*openapi.APIClient)

	request := map[string]interface{}{
		"name":        d.Get("name").(string),
		"description": d.Get("description").(string),
		"status":      d.Get("status").(string),
		"rules":       map[string]interface{}{"inbound": expandFirewallRules(d)},
	}
	var res struct {
		Data []firewallResponse `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodPost, "/v1/firewalls", nil, request, &res)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	if len(res.Data) != 1 {
		return MultipleDataObjectsError(nil)
	}
	firewallId := res.Data[0].FirewallId
	// Save the id first: if an assignment fails below, the firewall is kept
	// in state (as tainted) instead of being orphaned.
	d.SetId(firewallId)

	toAssign := setToInt64s(d.Get("instance_ids").(*schema.Set))
	if diags := assignFirewallInstances(ctx, client, firewallId, toAssign, d.Timeout(schema.TimeoutCreate)); diags.HasError() {
		return diags
	}

	return resourceFirewallRead(ctx, d, m)
}

func resourceFirewallRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	client := m.(*openapi.APIClient)

	firewall, httpResp, err := retrieveFirewall(ctx, client, d.Id())
	if err != nil {
		if removeFromStateIfNotFound(d, httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}
	return setFirewallData(*firewall, d)
}

func resourceFirewallUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	firewallId := d.Id()

	if d.HasChanges("name", "status", "description") {
		request := map[string]interface{}{}
		if d.HasChange("name") {
			request["name"] = d.Get("name").(string)
		}
		if d.HasChange("status") {
			request["status"] = d.Get("status").(string)
		}
		if d.HasChange("description") {
			request["description"] = d.Get("description").(string)
		}
		httpResp, err := apiRequest(ctx, client, http.MethodPatch, "/v1/firewalls/"+url.PathEscape(firewallId), nil, request, nil)
		if err != nil {
			return apiErrorDiags(httpResp, err)
		}
	}

	if d.HasChange("rules") {
		request := map[string]interface{}{
			"rules": map[string]interface{}{"inbound": expandFirewallRules(d)},
		}
		httpResp, err := apiRequest(ctx, client, http.MethodPut, "/v1/firewalls/"+url.PathEscape(firewallId), nil, request, nil)
		if err != nil {
			return apiErrorDiags(httpResp, err)
		}
	}

	if d.HasChange("instance_ids") {
		oldRaw, newRaw := d.GetChange("instance_ids")
		toAdd, toRemove := diffInt64Sets(
			setToInt64s(oldRaw.(*schema.Set)),
			setToInt64s(newRaw.(*schema.Set)),
		)
		for _, instanceId := range toRemove {
			httpResp, err := unassignFirewallInstance(ctx, client, firewallId, instanceId)
			if err != nil && !isNotFound(httpResp) {
				return apiErrorDiags(httpResp, err)
			}
		}
		if diags := assignFirewallInstances(ctx, client, firewallId, toAdd, d.Timeout(schema.TimeoutUpdate)); diags.HasError() {
			return diags
		}
	}

	return resourceFirewallRead(ctx, d, m)
}

func resourceFirewallDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	firewallId := d.Id()

	firewall, httpResp, err := retrieveFirewall(ctx, client, firewallId)
	if err != nil {
		if isNotFound(httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}

	for _, instanceId := range firewall.instanceIds() {
		httpResp, err := unassignFirewallInstance(ctx, client, firewallId, instanceId)
		if err != nil && !isNotFound(httpResp) {
			return apiErrorDiags(httpResp, err)
		}
	}

	httpResp, err = apiRequest(ctx, client, http.MethodDelete, "/v1/firewalls/"+url.PathEscape(firewallId), nil, nil, nil)
	if err != nil && !isNotFound(httpResp) {
		return apiErrorDiags(httpResp, err)
	}
	return nil
}

// assignFirewallInstances waits for each instance to be running, because the
// API rejects assignments to instances that are still being provisioned, and
// assigns it to the firewall.
func assignFirewallInstances(
	ctx context.Context,
	client *openapi.APIClient,
	firewallId string,
	instanceIds []int64,
	timeout time.Duration,
) diag.Diagnostics {
	for _, instanceId := range instanceIds {
		if _, diags := waitForInstanceReady(ctx, client, instanceId, timeout); diags.HasError() {
			return diags
		}
		httpResp, err := apiRequest(ctx, client, http.MethodPost,
			fmt.Sprintf("/v1/firewalls/%s/instances/%d", url.PathEscape(firewallId), instanceId), nil, nil, nil)
		if err != nil {
			return apiErrorDiags(httpResp, err)
		}
	}
	return nil
}

func unassignFirewallInstance(
	ctx context.Context,
	client *openapi.APIClient,
	firewallId string,
	instanceId int64,
) (*http.Response, error) {
	return apiRequest(ctx, client, http.MethodDelete,
		fmt.Sprintf("/v1/firewalls/%s/instances/%d", url.PathEscape(firewallId), instanceId), nil, nil, nil)
}

func setFirewallData(firewall firewallResponse, d *schema.ResourceData) diag.Diagnostics {
	d.SetId(firewall.FirewallId)
	values := map[string]interface{}{
		"name":             firewall.Name,
		"status":           firewall.Status,
		"description":      firewall.Description,
		"instance_ids":     firewall.instanceIds(),
		"rules":            flattenFirewallRules(firewall.Rules.Inbound),
		"instances_status": flattenFirewallInstancesStatus(firewall.InstanceStatus),
		"created_date":     firewall.CreatedDate.Format(time.RFC850),
	}
	for key, value := range values {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("could not set %s: %s", key, err)
		}
	}
	return nil
}

func flattenFirewallInstancesStatus(statuses []firewallInstanceStatus) []interface{} {
	out := make([]interface{}, 0, len(statuses))
	for _, status := range statuses {
		errorMessage := ""
		if status.ErrorMessage != nil {
			errorMessage = *status.ErrorMessage
		}
		out = append(out, map[string]interface{}{
			"instance_id":   int(status.InstanceId),
			"status":        status.Status,
			"error_message": errorMessage,
		})
	}
	return out
}

// implicitBlockAllDisplayName names the rule the API appends to every
// firewall: drop everything that no earlier rule accepted.
const implicitBlockAllDisplayName = "Block all traffic"

// isImplicitBlockAllRule reports whether the rule is the drop-all rule the
// API appends. The provider hides it, because it is not part of the
// configuration and the API adds it again after every change.
func isImplicitBlockAllRule(rule firewallRule) bool {
	return rule.DisplayName == implicitBlockAllDisplayName &&
		strings.EqualFold(rule.Action, "drop") &&
		rule.Protocol == "" &&
		len(rule.DestPorts) == 0 &&
		len(rule.SrcCidr.Ipv4) == 0 &&
		len(rule.SrcCidr.Ipv6) == 0
}

func flattenFirewallRules(inbound []firewallRule) []interface{} {
	if n := len(inbound); n > 0 && isImplicitBlockAllRule(inbound[n-1]) {
		inbound = inbound[:n-1]
	}
	rules := make([]interface{}, 0, len(inbound))
	for _, rule := range inbound {
		cidr := map[string]interface{}{
			"ipv4": stringsToSet(rule.SrcCidr.Ipv4),
			"ipv6": stringsToSet(rule.SrcCidr.Ipv6),
		}
		protocol := rule.Protocol
		if protocol == "" {
			protocol = firewallProtocolAny
		}
		rules = append(rules, map[string]interface{}{
			"protocol":     protocol,
			"action":       rule.Action,
			"status":       rule.Status,
			"display_name": rule.DisplayName,
			"dest_ports":   stringsToSet(rule.DestPorts),
			"src_cidr":     []interface{}{cidr},
		})
	}
	return []interface{}{map[string]interface{}{inboundRuleKey: rules}}
}

func stringsToSet(values []string) *schema.Set {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return schema.NewSet(schema.HashString, items)
}

func setToInt64s(set *schema.Set) []int64 {
	if set == nil {
		return nil
	}
	ids := make([]int64, 0, set.Len())
	for _, item := range set.List() {
		ids = append(ids, int64(item.(int)))
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// diffInt64Sets returns the ids only in newIds and the ids only in oldIds.
func diffInt64Sets(oldIds []int64, newIds []int64) (added []int64, removed []int64) {
	oldSet := map[int64]bool{}
	for _, id := range oldIds {
		oldSet[id] = true
	}
	newSet := map[int64]bool{}
	for _, id := range newIds {
		newSet[id] = true
		if !oldSet[id] {
			added = append(added, id)
		}
	}
	for _, id := range oldIds {
		if !newSet[id] {
			removed = append(removed, id)
		}
	}
	return added, removed
}
