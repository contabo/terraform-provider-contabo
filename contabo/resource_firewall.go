package contabo

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

var httpConflict409 string = "409 Conflict"

const inboundRuleKey = "inbound"
const firewallNetworkAddOnId int64 = 1501

type jmap map[string]interface{}

func resourceFirewall() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallCreate,
		UpdateContext: resourceFirewallUpdate,
		ReadContext:   resourceFirewallRead,
		DeleteContext: resourceFirewallDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceContaboFirewallImport,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"created_date": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The creation date of the Firewall.",
			},
			"id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the Firewall. Use it to manage it!",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Firewall. It may contain letters, numbers, colons, dashes, and underscores. There is a limit of 255 characters per Firewall name.",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Firewall. There is a limit of 255 characters per Firewall.",
			},
			"status": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Status of the Firewall. It can be `active`, or `inactive`.",
			},
			"instance_ids": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Optional:    true,
				Description: "Add the instace Ids to the firewall here. If you do not add any instance Ids an empty firewall will be created.",
			},
			"instances_status": {
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				Description: "The status of every instance in the firewall",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The instance",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the instance.",
						},
						"error_message": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional:    true,
							Description: "Status of the instance.",
						},
					},
				},
			},
			"rules": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"inbound": {
							Type:        schema.TypeList,
							Computed:    true,
							Optional:    true,
							Description: "Inbound rules for this Firewall",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"protocol": {
										Type:        schema.TypeString,
										Computed:    true,
										Optional:    true,
										Description: "Define the protocol for the rule. Allowed protocols are `tcp`, `udp` and `icmp`.",
									},
									"action": {
										Type:        schema.TypeString,
										Computed:    true,
										Optional:    true,
										Description: "Action of the rule, currently there is just `accept`.",
									},
									"status": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Status of the rule. It can be `active`, or `inactive`.",
									},
									"dest_ports": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"src_cidr": {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ipv4": &schema.Schema{
													Type:        schema.TypeSet,
													Elem:        &schema.Schema{Type: schema.TypeString},
													Optional:    true,
													Description: "Provide allowed IPv4 addresses as string array for this rule",
												},
												"ipv6": &schema.Schema{
													Type:        schema.TypeSet,
													Elem:        &schema.Schema{Type: schema.TypeString},
													Optional:    true,
													Description: "Provide allowed IPv6 addresses as string array for this rule",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceFirewallCreate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	status := d.Get("status").(string)
	createFirewallRequest := openapi.NewCreateFirewallRequestWithDefaults()
	createFirewallRequest.Name = name
	createFirewallRequest.Description = &description
	createFirewallRequest.Status = status
	createFirewallRequest.Rules = openapi.NewRulesRequestWithDefaults()
	createFirewallRequest.Rules.Inbound = make([]openapi.FirewallRuleRequest, 0)
	rules := getFirewallInboundRules(d)

	if len(rules) > 0 {
		for _, rule := range rules {
			inboundRuleMap := rule.(map[string]interface{})
			if inboundRuleMap["action"].(string) != "" && inboundRuleMap["status"].(string) != "" {
				protocol := inboundRuleMap["protocol"].(string)
				if strings.EqualFold(protocol, "any") {
					protocol = ""
				}

				action := inboundRuleMap["action"].(string)
				status := inboundRuleMap["status"].(string)
				destPorts := getDestPorts(inboundRuleMap)
				srcCidr := *openapi.NewSrcCidrWithDefaults()
				srcCidr.SetIpv4(getSrcCidrIpv4Addresses(inboundRuleMap))
				srcCidr.SetIpv6(getSrcCidrIpv6Addresses(inboundRuleMap))
				apiInboundRule := openapi.NewFirewallRuleRequest(protocol, destPorts, srcCidr, action, status)
				createFirewallRequest.Rules.Inbound = append(createFirewallRequest.Rules.Inbound, *apiInboundRule)
			}
		}
	}
	res, httpResp, err := client.FirewallsApi.
		CreateFirewall(context.Background()).
		XRequestId(uuid.NewV4().String()).
		CreateFirewallRequest(*createFirewallRequest).
		Execute()
	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	if len(res.Data) != 1 {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Internal Error: should have returned only one object",
		})
	}

	instancesToAdd := d.Get("instance_ids").(*schema.Set).List()
	firewallId := res.Data[0].FirewallId

	for _, instanceId := range instancesToAdd {
		instanceIdInt := instanceId.(int)
		instanceId := int64(instanceIdInt)

		pollInstance(diags, client, instanceId)

		httpResp, err = assignInstanceToFirewall(diags, client, firewallId, instanceId)
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}
	}

	d.SetId(res.Data[0].FirewallId)
	return resourceFirewallRead(ctx, d, m)
}

func getFirewallInboundRules(d *schema.ResourceData) []interface{} {
	rules := d.Get("rules").([]interface{})

	if rules != nil && len(rules) > 0 {
		rslt := rules[0].(map[string]interface{})[inboundRuleKey].([]interface{})
		return rslt
	}
	return nil
}

func getSrcCidrIpv4Addresses(inboundRuleMap map[string]interface{}) []string {
	srcCidrs := inboundRuleMap["src_cidr"].([]interface{})
	ipv4AddressesStrArr := make([]string, 0)

	for _, srcCidr := range srcCidrs {
		ipv4Addresses := getIpv4Addresses(srcCidr)
		for _, ipv4 := range ipv4Addresses {
			ipv4AddressesStrArr = append(ipv4AddressesStrArr, ipv4.(string))
		}
	}
	return ipv4AddressesStrArr
}

func getSrcCidrIpv6Addresses(inboundRuleMap map[string]interface{}) []string {
	srcCidrs := inboundRuleMap["src_cidr"].([]interface{})

	ipv6AddressesStrArr := make([]string, 0)

	for _, srcCidr := range srcCidrs {
		ipv6Addresses := getIpv6Addresses(srcCidr)
		for _, ipv6 := range ipv6Addresses {
			ipv6AddressesStrArr = append(ipv6AddressesStrArr, ipv6.(string))
		}
	}
	return ipv6AddressesStrArr
}

func handleFirewallInstanceChanges(diags diag.Diagnostics,
	d *schema.ResourceData,
	client *openapi.APIClient,
	firewallId string) diag.Diagnostics {

	// Remove instances which are not more in this firewall
	old, new := d.GetChange("instance_ids")
	oldInstanceIds := old.(*schema.Set).List()
	for _, instanceId := range oldInstanceIds {
		instanceIdInt := instanceId.(int)
		instanceId := int64(instanceIdInt)

		httpResp, err := unassignInstanceToFirewall(diags, client, firewallId, instanceId)
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}
	}

	// Add new instances which are now in this firewall
	newInstanceIds := new.(*schema.Set).List()
	for _, instanceId := range newInstanceIds {
		instanceIdInt := instanceId.(int)
		instanceId := int64(instanceIdInt)

		httpResp, err := assignInstanceToFirewall(diags, client, firewallId, instanceId)
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}
	}
	return nil
}

func getDestPorts(
	inboundRuleMap map[string]interface{}) []string {

	destPorts := inboundRuleMap["dest_ports"].(*schema.Set).List()
	destPortsStringArr := make([]string, 0)
	if len(destPorts) > 0 {
		for _, srcPort := range destPorts {
			destPortsStringArr = append(destPortsStringArr, srcPort.(string))
		}
	}
	return destPortsStringArr
}

func getIpv4Addresses(srcCidr interface{}) []interface{} {
	return srcCidr.(map[string]interface{})["ipv4"].(*schema.Set).List()
}

func getIpv6Addresses(srcCidr interface{}) []interface{} {
	return srcCidr.(map[string]interface{})["ipv6"].(*schema.Set).List()
}

func resourceFirewallRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	firewallId := d.Id()

	res, httpResp, err := client.FirewallsApi.
		RetrieveFirewall(ctx, firewallId).
		XRequestId(uuid.NewV4().String()).
		Execute()
	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}
	if len(res.Data) != 1 {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Internal Error: should have returned only one object",
		})
	}
	resultDiags, _ := AddFirewallToData(res.Data[0], d, diags)
	return resultDiags
}

func resourceFirewallUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	firewallId := d.Id()

	if d.HasChange("instance_ids") {
		rsltDiag := handleFirewallInstanceChanges(diags, d, client, firewallId)
		if rsltDiag != nil {
			return rsltDiag
		}
	}

	if d.HasChange("rules") {
		rsltDiag := handleFirewallRulesChanges(diags, d, client, firewallId)
		if rsltDiag != nil {
			return rsltDiag
		}
	}

	var updateFirewallRequest openapi.PatchFirewallRequest
	anyChange := false
	if d.HasChange("name") {
		firewallName := d.Get("name").(string)
		updateFirewallRequest.Name = &firewallName
		anyChange = true
	}
	if d.HasChange("status") {
		firewallStatus := d.Get("status").(string)
		updateFirewallRequest.Status = &firewallStatus
		anyChange = true
	}
	if d.HasChange("description") {
		firewallDesc := d.Get("description").(string)
		updateFirewallRequest.Description = &firewallDesc
		anyChange = true
	}

	if anyChange {
		_, httpResp, err := client.FirewallsApi.
			PatchFirewall(context.Background(), firewallId).
			XRequestId(uuid.NewV4().String()).
			PatchFirewallRequest(updateFirewallRequest).Execute()
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}

		d.Set("updated_at", time.Now().Format(time.RFC850))
		return resourceFirewallRead(ctx, d, m)
	}
	return diags
}

func handleFirewallRulesChanges(
	diags diag.Diagnostics,
	d *schema.ResourceData,
	client *openapi.APIClient,
	firewallId string) diag.Diagnostics {
	rules := getFirewallInboundRules(d)
	firewallRulesRequest := *openapi.NewPutFirewallRequestWithDefaults()
	firewallRulesRequest.Rules = openapi.NewRulesRequestWithDefaults()
	firewallRulesRequest.Rules.Inbound = make([]openapi.FirewallRuleRequest, 0)

	if len(rules) > 0 {
		for _, rule := range rules {
			inboundRuleMap := rule.(map[string]interface{})
			if inboundRuleMap["action"].(string) != "" && inboundRuleMap["status"].(string) != "" {
				protocol := inboundRuleMap["protocol"].(string)
				if strings.EqualFold(protocol, "any") {
					protocol = ""
				}

				action := inboundRuleMap["action"].(string)
				status := inboundRuleMap["status"].(string)
				destPorts := getDestPorts(inboundRuleMap)
				srcCidr := *openapi.NewSrcCidrWithDefaults()
				srcCidr.SetIpv4(getSrcCidrIpv4Addresses(inboundRuleMap))
				srcCidr.SetIpv6(getSrcCidrIpv6Addresses(inboundRuleMap))
				apiInboundRule := openapi.NewFirewallRuleRequest(protocol, destPorts, srcCidr, action, status)
				firewallRulesRequest.Rules.Inbound = append(firewallRulesRequest.Rules.Inbound, *apiInboundRule)
			}
		}
	}
	_, httpResp, err := client.FirewallsApi.
		PutFirewall(context.Background(), firewallId).
		XRequestId(uuid.NewV4().String()).
		PutFirewallRequest(firewallRulesRequest).
		Execute()
	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}
	return nil
}

func resourceFirewallDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	firewallId := d.Id()

	readRes, httpResp, err := client.FirewallsApi.
		RetrieveFirewall(ctx, firewallId).
		XRequestId(uuid.NewV4().String()).
		Execute()
	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	for _, i := range readRes.Data[0].Instances {
		client.FirewallsApi.UnassignInstanceFirewall(ctx, firewallId, i.InstanceId).XRequestId(uuid.NewV4().String()).Execute()
	}

	httpResp, err = client.FirewallsApi.
		DeleteFirewall(ctx, firewallId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}
	d.SetId("")
	return diags
}

func assignInstanceToFirewall(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	firewallId string,
	instanceId int64) (*http.Response, error) {

	_, httpResp, err := client.FirewallsApi.AssignInstanceFirewall(
		context.Background(),
		firewallId,
		instanceId).XRequestId(uuid.NewV4().String()).Execute()

	return httpResp, err
}

func unassignInstanceToFirewall(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	firewallId string,
	instanceId int64) (*http.Response, error) {

	_, httpResp, err := client.FirewallsApi.UnassignInstanceFirewall(
		context.Background(),
		firewallId,
		instanceId).XRequestId(uuid.NewV4().String()).Execute()
	return httpResp, err
}

func resourceContaboFirewallImport(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{}) ([]*schema.ResourceData, error) {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	firewallId := d.Id()

	res, httpResp, err := client.FirewallsApi.
		RetrieveFirewall(ctx, firewallId).
		XRequestId(uuid.NewV4().String()).
		Execute()
	if err != nil {
		HandleResponseErrors(diags, httpResp)
		return nil, err
	}
	if len(res.Data) != 1 {
		return nil, fmt.Errorf("should have returned only one object: %v", err)
	}

	_, resourceData := AddFirewallToData(res.Data[0], d, diags)
	return []*schema.ResourceData{resourceData}, nil
}

func AddFirewallToData(
	firewall openapi.FirewallResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) (diag.Diagnostics, *schema.ResourceData) {
	id := firewall.FirewallId
	if err := d.Set("id", id); err != nil {
		return diag.FromErr(err), nil
	}
	if err := d.Set("name", firewall.Name); err != nil {
		return diag.FromErr(err), nil
	}
	if err := d.Set("status", firewall.Status); err != nil {
		return diag.FromErr(err), nil
	}
	if err := d.Set("description", firewall.Description); err != nil {
		return diag.FromErr(err), nil
	}
	var instanceIds []int64
	for _, instance := range firewall.Instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	if err := d.Set("instance_ids", instanceIds); err != nil {
		return diag.FromErr(err), nil
	}
	rules := buildFirewallRules(&firewall.Rules)
	if err := d.Set("rules", rules); err != nil {
		return diag.FromErr(err), nil
	}
	var instancesStatus []interface{}

	for _, instanceStatus := range firewall.InstanceStatus {
		newStatus := make(map[string]interface{})
		newStatus["instance_id"] = instanceStatus.InstanceId
		newStatus["status"] = instanceStatus.Status
		newStatus["error_message"] = instanceStatus.ErrorMessage

		instancesStatus = append(instancesStatus, newStatus)
	}

	if err := d.Set("instances_status", instancesStatus); err != nil {
		return diag.FromErr(err), nil
	}
	createdAt := firewall.CreatedDate.Format(time.RFC850)
	if err := d.Set("created_date", createdAt); err != nil {
		return diag.FromErr(err), nil
	}
	return diags, d
}

func buildFirewallRules(rulesResponse *openapi.Rules) []interface{} {
	if rulesResponse != nil {
		var sliceOfRules = make([]interface{}, 0)
		var ruleList = make([]interface{}, 0)

		inboundRules := make(map[string]interface{}, 0)
		for _, ruleInbound := range rulesResponse.Inbound {
			rule := make(map[string]interface{})

			rule["protocol"] = ruleInbound.Protocol
			rule["dest_ports"] = ruleInbound.DestPorts
			rule["status"] = ruleInbound.Status
			rule["action"] = ruleInbound.Action

			var srcCidrs []interface{}
			srcCidrMap := make(map[string]interface{})
			if ruleInbound.SrcCidr.Ipv4 != nil {
				srcCidrMap["ipv4"] = buildIps(*ruleInbound.SrcCidr.Ipv4)
			}
			if ruleInbound.SrcCidr.Ipv6 != nil {
				srcCidrMap["ipv6"] = buildIps(*ruleInbound.SrcCidr.Ipv6)
			}
			srcCidrs = append(srcCidrs, srcCidrMap)
			rule["src_cidr"] = srcCidrs

			ruleList = append(ruleList, rule)
		}
		inboundRules["inbound"] = ruleList
		sliceOfRules = append(sliceOfRules, inboundRules)

		return sliceOfRules
	}
	return nil
}

func buildIps(ips []string) *schema.Set {
	interfaceIp := make([]interface{}, len(ips))
	if !(len(ips) > 0) {
		return schema.NewSet(schema.HashString, interfaceIp)
	}
	for i := range ips {
		interfaceIp[i] = ips[i]
	}
	return schema.NewSet(schema.HashString, interfaceIp)
}
