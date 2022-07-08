package contabo

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

var privateNetworkAddOnId int64 = 1477
var httpConflict string = "409 Conflict"

func resourcePrivateNetwork() *schema.Resource {
	return &schema.Resource{
		Description:   "Provides a Contabo [Private Network](https://api.contabo.com/#tag/Private-Networks) resource.  Private Networks can contain your compute instances whereby they are able to communicate with each other in full usolation, using private IP addresses.",
		CreateContext: resourcePrivateNetworkCreate,
		ReadContext:   resourcePrivateNetworkRead,
		UpdateContext: resourcePrivateNetworkUpdate,
		DeleteContext: resourcePrivateNetworkDelete,
		Schema: map[string]*schema.Schema{
			"created_date": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The creation date of the Private Network.",
			},
			"updated_at": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Time of the last update of the private network.",
			},
			"id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the Private Network. Use it to manage it!",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the Private Network. It may contain letters, numbers, colons, dashes, and underscores. There is a limit of 255 characters per Private Network name.",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Private Network. There is a limit of 255 characters per Private Network.",
			},
			"instance_ids": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Optional:    true,
				Description: "Add the instace Ids to the private network here. If you do not add any instance Ids an empty private network will be created.",
			},
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "EU",
				Description: "The region where the Private Network should be located. Default region is the EU.",
			},
			"region_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the region where the Private Network is located.",
			},
			"data_center": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The specific data center where the Private Network is located.",
			},
			"available_ips": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The totality of available IPs in the Private Network.",
			},
			"cidr": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The cidr range of the Private Network.",
			},
		},
	}
}

func resourcePrivateNetworkCreate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	privateNetworkName := d.Get("name").(string)
	privateNetworkDescription := d.Get("description").(string)
	privateNetworkRegion := d.Get("region").(string)

	createPrivateNetworkRequest := openapi.NewCreatePrivateNetworkRequestWithDefaults()
	createPrivateNetworkRequest.Name = privateNetworkName
	createPrivateNetworkRequest.Description = &privateNetworkDescription
	createPrivateNetworkRequest.Region = privateNetworkRegion

	res, httpResp, err := client.PrivateNetworksApi.
		CreatePrivateNetwork(context.Background()).
		XRequestId(uuid.NewV4().String()).
		CreatePrivateNetworkRequest(*createPrivateNetworkRequest).
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
	privateNetworkId := res.Data[0].PrivateNetworkId

	for _, instanceId := range instancesToAdd {
		instanceIdInt := instanceId.(int)
		instanceId := int64(instanceIdInt)

		httpResp, err := addPrivateNetworkAddOnToInstance(diags, client, instanceId)
		isErrNotConflict := !strings.Contains(err.Error(), httpConflict)
		if err != nil && isErrNotConflict {
			return HandleResponseErrors(diags, httpResp)
		}

		httpResp, err = assignInstanceToPrivateNetwork(diags, client, privateNetworkId, instanceId)
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}
	}
	d.SetId(strconv.Itoa(int(privateNetworkId)))
	return resourcePrivateNetworkRead(ctx, d, m)
}

func assignInstanceToPrivateNetwork(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	privateNetworkId,
	instanceId int64) (*http.Response, error) {

	_, httpResp, err := client.PrivateNetworksApi.AssignInstancePrivateNetwork(
		context.Background(),
		privateNetworkId,
		instanceId).XRequestId(uuid.NewV4().String()).Execute()

	return httpResp, err
}

func unassignInstanceToPrivateNetwork(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	privateNetworkId int64,
	instanceId int64) (*http.Response, error) {

	_, httpResp, err := client.PrivateNetworksApi.UnassignInstancePrivateNetwork(
		context.Background(),
		privateNetworkId,
		instanceId).XRequestId(uuid.NewV4().String()).Execute()

	return httpResp, err
}

func addPrivateNetworkAddOnToInstance(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	instanceId int64) (*http.Response, error) {

	var upgradeInstance openapi.UpgradeInstanceRequest
	upgradeInstance.AddOns = []int64{privateNetworkAddOnId}

	_, httpResp, err := client.InstancesApi.UpgradeInstance(context.Background(), instanceId).XRequestId(uuid.NewV4().String()).
		UpgradeInstanceRequest(upgradeInstance).
		Execute()
	return httpResp, err
}

func resourcePrivateNetworkRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	privateNetworkId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.PrivateNetworksApi.
		RetrievePrivateNetwork(ctx, privateNetworkId).
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

	return AddPrivateNetworkToData(res.Data[0], d, diags)
}

func resourcePrivateNetworkUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	privateNetworkId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	updatePrivateNetworkRequest := openapi.NewPatchPrivateNetworkRequest()
	anyChange := false

	if d.HasChange("name") {
		privateNetworkName := d.Get("name").(string)
		updatePrivateNetworkRequest.Name = &privateNetworkName
		anyChange = true
	}

	if d.HasChange("description") {
		description := d.Get("description").(string)
		updatePrivateNetworkRequest.Description = &description
		anyChange = true
	}

	if d.HasChange("instance_ids") {
		rsltDiag := handleInstanceChanges(diags, d, client, privateNetworkId)
		if rsltDiag != nil {
			return rsltDiag
		}
	}

	if anyChange {
		_, httpResp, err := client.PrivateNetworksApi.
			PatchPrivateNetwork(context.Background(), privateNetworkId).
			XRequestId(uuid.NewV4().String()).
			PatchPrivateNetworkRequest(*updatePrivateNetworkRequest).
			Execute()

		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}

		d.Set("updated_at", time.Now().Format(time.RFC850))
		return resourcePrivateNetworkRead(ctx, d, m)
	}
	return diags
}

func handleInstanceChanges(diags diag.Diagnostics,
	d *schema.ResourceData,
	client *openapi.APIClient,
	privateNetworkId int64) diag.Diagnostics {

	//Remove instances which are not more in this private network
	old, new := d.GetChange("instance_ids")
	oldInstanceIds := old.(*schema.Set).List()
	for _, instanceId := range oldInstanceIds {
		instanceIdInt := instanceId.(int)
		instanceId := int64(instanceIdInt)

		httpResp, err := unassignInstanceToPrivateNetwork(diags, client, privateNetworkId, instanceId)
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}
	}

	//Add new instances which are now in this private network
	newInstanceIds := new.(*schema.Set).List()
	for _, instanceId := range newInstanceIds {
		instanceIdInt := instanceId.(int)
		instanceId := int64(instanceIdInt)

		httpResp, err := addPrivateNetworkAddOnToInstance(diags, client, instanceId)
		isErrNotConflict := !strings.Contains(err.Error(), httpConflict)
		if err != nil && isErrNotConflict {
			return HandleResponseErrors(diags, httpResp)
		}

		httpResp, err = assignInstanceToPrivateNetwork(diags, client, privateNetworkId, instanceId)
		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		}
	}
	return nil
}

func resourcePrivateNetworkDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	privateNetworkId, err := strconv.ParseInt(d.Id(), 10, 64)

	if err != nil {
		return diag.FromErr(err)
	}

	readRes, httpResp, err := client.PrivateNetworksApi.
		RetrievePrivateNetwork(ctx, privateNetworkId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	for _, i := range readRes.Data[0].Instances {
		client.PrivateNetworksApi.UnassignInstancePrivateNetwork(ctx, privateNetworkId, i.InstanceId).XRequestId(uuid.NewV4().String()).Execute()
	}

	httpResp, err = client.PrivateNetworksApi.
		DeletePrivateNetwork(ctx, privateNetworkId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")

	return diags
}

func AddPrivateNetworkToData(
	privateNetwork openapi.PrivateNetworkResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	id := strconv.Itoa(int(privateNetwork.PrivateNetworkId))
	if err := d.Set("id", id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", privateNetwork.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", privateNetwork.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region", privateNetwork.Region); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("data_center", privateNetwork.DataCenter); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("available_ips", privateNetwork.AvailableIps); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cidr", privateNetwork.Cidr); err != nil {
		return diag.FromErr(err)
	}
	createdDate := privateNetwork.CreatedDate.Format(time.RFC850)
	if err := d.Set("created_date", createdDate); err != nil {
		return diag.FromErr(err)
	}
	var instanceIds []int64
	for _, instance := range privateNetwork.Instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	if err := d.Set("instance_ids", instanceIds); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
