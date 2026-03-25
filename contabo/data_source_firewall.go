package contabo

import (
	"context"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourceFirewall() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFirewallRead,
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

func dataSourceFirewallRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	firewallId := d.Get("id").(string)

	res, httpResp, err := client.FirewallsApi.
		RetrieveFirewall(ctx, firewallId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(res.Data[0].FirewallId)

	resultDiags, _ := AddFirewallToData(res.Data[0], d, diags)
	return resultDiags
}
