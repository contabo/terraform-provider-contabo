package contabo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// instanceUpgradeAddOns maps the add-on ids that can be added to an existing
// instance to the attribute of the upgrade request that orders them. The ids
// are the catalog group ids that the API reports in addOns[].id.
var instanceUpgradeAddOns = map[int64]string{
	1477: "privateNetworking",
	2214: "backup",
}

// Attributes whose change reinstalls the instance.
var instanceReinstallAttributes = []string{
	"image_id", "ssh_keys", "root_password", "user_data", "default_user", "application_id",
}

func instanceIpSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address.",
			},
			"netmask_cidr": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the network mask in CIDR notation, for example `24`.",
			},
			"gateway": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address of the gateway.",
			},
		},
	}
}

func instanceIpConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The main IPv4 address and the main IPv6 address of the instance, with the network settings of the addresses.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"v4": {Type: schema.TypeList, Computed: true, Elem: instanceIpSchema()},
				"v6": {Type: schema.TypeList, Computed: true, Elem: instanceIpSchema()},
			},
		},
	}
}

func instanceAdditionalIpsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The additional IP addresses of the instance.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"v4": {Type: schema.TypeList, Computed: true, Elem: instanceIpSchema()},
			},
		},
	}
}

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource manages a compute instance, which is a VPS or a VDS. When Terraform creates an instance, Contabo orders the instance and starts to bill you for the instance. When Terraform destroys an instance, Contabo cancels the instance at the end of the contract period. You can turn off the cancellation with `cancel_on_destroy = false`. The default images support [cloud-init](https://cloud-init.io/) through the `user_data` setting. Custom images need cloud-init packages for this setting to work.",
		CreateContext: resourceInstanceCreate,
		ReadContext:   resourceInstanceRead,
		UpdateContext: resourceInstanceUpdate,
		DeleteContext: resourceInstanceDelete,
		CustomizeDiff: resourceInstanceCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: resourceInstanceImport,
		},
		// Version 1 adds cancel_on_destroy. States written by the official
		// provider or earlier versions of this one have no value for it.
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceInstanceV0Type(),
				Upgrade: resourceInstanceStateUpgradeV0,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: resourceInstanceSchema(),
	}
}

func resourceInstanceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The identifier of the compute instance. Use it to manage it!",
		},
		"existing_instance_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Optional:    true,
			Description: "The id of an existing instance. When you set this value, the provider manages the existing instance instead of ordering a new instance. During the adoption, the provider sets the `display_name` value and orders the add-ons, but the provider does not reinstall the instance. On later runs, a change of a reinstall setting reinstalls the instance as usual. You can also use `terraform import` instead of this setting. If destroying the resource must not cancel the instance, set `cancel_on_destroy = false`.",
		},
		"cancel_on_destroy": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Whether destroying the resource cancels the instance. When the value is `false`, destroying the resource only removes the instance from the Terraform state, and the instance keeps running and keeps costing money. The default value is `true`.",
		},
		"last_updated": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Time of the last update of the compute instance.",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Name of the compute instance.",
		},
		"display_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The display name of the instance. The Customer Control Panel shows this name.",
		},
		"image_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "CAUTION: When you change this value, the provider reinstalls the server. The id of the image that the instance runs. When you do not set this value, the API uses its default image. You can find image ids with the `contabo_image` data source, with the [API](https://api.contabo.com/#tag/Images/operation/retrieveImage), or with the command `cntb get images`.",
		},
		"region": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The region of the instance, for example `EU`, `UK`, `US-central`, `US-east`, `US-west`, `SIN`, `AUS`, `JPN` or `IND`. The default value is `EU`. You cannot change the region after the instance exists.",
		},
		"region_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The full name of the region of the instance, for example `European Union`.",
		},
		"data_center": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The data center of the instance, for example `European Union 2`.",
		},
		"product_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The product of the instance, for example `V153`. The [API documentation](https://api.contabo.com/#tag/Instances/operation/createInstance) lists the current products. The API cannot change the product of an existing instance. To change the product, upgrade the instance in the Customer Control Panel, and then set the new product id in your configuration.",
		},
		"product_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the product of the instance, for example `Cloud VPS 4`.",
		},
		"ip_config": instanceIpConfigSchema(),
		"mac_address": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Mac address of the instance.",
		},
		"ram_mb": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Image ram size in megabyte.",
		},
		"cpu_cores": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "CPU core count of the instance.",
		},
		"disk_mb": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Image disk size of the instance in megabyte.",
		},
		"os_type": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Type of operating system (OS) installed on the instance.",
		},
		"ssh_keys": {
			Computed: true,
			Optional: true,
			Type:     schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeInt,
			},
			Description: "CAUTION: When you change this value, the provider reinstalls the server. The secret ids of the public SSH keys that the installation adds for the default user. The keys work on Linux and BSD systems. See the `contabo_secret` resource.",
		},
		"root_password": {
			Optional:    true,
			Type:        schema.TypeInt,
			Description: "CAUTION: When you change this value, the provider reinstalls the server. The secret id of the root password of the instance.",
		},
		"created_date": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The creation date of the compute instance.",
		},
		"cancel_date": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The date when the instance will be cancelled. Setting this value has no effect, so treat the value as read-only.",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The status of the instance, for example `provisioning`, `installing`, `running`, `stopped`, `error` or `uninstalled`.",
		},
		"v_host_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Identifier of the host system.",
		},
		"add_ons": {
			Type:        schema.TypeList,
			Computed:    true,
			Optional:    true,
			Description: "The add-ons of the instance. When Terraform creates the instance, the provider orders every add-on in this list together with the instance. For an existing instance, the provider can add private networking (`1477`) and automated backup (`2214`) through the upgrade API. Other add-ons can only be ordered together with a new instance or in the Customer Control Panel. Removing an add-on from this list does not remove the add-on from the instance. When you set this list, the state only shows the add-ons from your configuration. The `contabo_instance` data source shows all add-ons.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Computed:    true,
						Optional:    true,
						Description: "The id of the add-on. The [product list](https://contabo.com/en/product-list/?show_ids=true) shows the ids.",
					},
					"quantity": {
						Type:        schema.TypeInt,
						Computed:    true,
						Optional:    true,
						Description: "The number of units of the add-on that you want.",
					},
				},
			},
		},
		"error_message": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The error message of the instance. The value is only set when the status of the instance is `error`.",
		},
		"product_type": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The category of the product of the instance, for example `ssd`, `nvme` or `vds`.",
		},
		"user_data": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "CAUTION: When you change this value, the provider reinstalls the server. The cloud-init configuration that the instance runs on the first start. The API does not return this value, so Terraform cannot see changes that someone makes outside of Terraform.",
		},
		"license": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "An extra software license for the product, for example for cPanel or Plesk. Windows does not need this license. The [API documentation](https://api.contabo.com/#tag/Instances/operation/createInstance) lists the licenses. The provider only uses this value when the provider creates the instance.",
		},
		"default_user": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringInSlice([]string{"root", "admin", "administrator"}, false),
			Description:  "CAUTION: When you change this value, the provider reinstalls the server. The name of the user that the installation creates with administrator rights. On Linux and BSD, the value is `admin` or `root`. The `admin` user uses sudo to get root rights. On Windows, the value is `admin` or `administrator`.",
		},
		"application_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "CAUTION: When you change this value, the provider reinstalls the server. The id of an application that the provider installs together with the image.",
		},
		"period": {
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "The first contract period in months, for example 1, 3, 6 or 12. The default value is 1 month. The provider only uses this value when the provider creates the instance.",
		},
		"additional_ips": instanceAdditionalIpsSchema(),
	}
}

func resourceInstanceImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := d.Set("cancel_on_destroy", true); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

// resourceInstanceV0Type is the schema of version 0. The upgrader only adds
// an attribute, so the current schema without it describes version 0.
func resourceInstanceV0Type() cty.Type {
	v0 := resourceInstanceSchema()
	delete(v0, "cancel_on_destroy")
	return (&schema.Resource{Schema: v0}).CoreConfigSchema().ImpliedType()
}

func resourceInstanceStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	if rawState == nil {
		rawState = map[string]interface{}{}
	}
	if _, ok := rawState["cancel_on_destroy"]; !ok {
		rawState["cancel_on_destroy"] = true
	}
	return rawState, nil
}

func resourceInstanceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)

	if existingId := d.Get("existing_instance_id").(string); existingId != "" {
		return adoptInstance(ctx, d, m, existingId)
	}

	request := map[string]interface{}{}
	setIfNotEmpty(request, "displayName", d.Get("display_name").(string))
	setIfNotEmpty(request, "imageId", d.Get("image_id").(string))
	setIfNotEmpty(request, "region", d.Get("region").(string))
	setIfNotEmpty(request, "productId", d.Get("product_id").(string))
	setIfNotEmpty(request, "userData", d.Get("user_data").(string))
	setIfNotEmpty(request, "license", d.Get("license").(string))
	setIfNotEmpty(request, "defaultUser", d.Get("default_user").(string))
	setIfNotEmpty(request, "applicationId", d.Get("application_id").(string))
	if sshKeys := interfacesToInt64s(d.Get("ssh_keys").([]interface{})); len(sshKeys) > 0 {
		request["sshKeys"] = sshKeys
	}
	if rootPassword := d.Get("root_password").(int); rootPassword > 0 {
		request["rootPassword"] = rootPassword
	}
	period := d.Get("period").(int)
	if period == 0 {
		period = 1
	}
	request["period"] = period
	if addOns := expandInstanceAddOns(d.Get("add_ons").([]interface{})); len(addOns) > 0 {
		request["addOns"] = map[string]interface{}{"addonsIds": addOns}
	}

	startedAt := time.Now()
	var res struct {
		Data []struct {
			InstanceId int64 `json:"instanceId"`
		} `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodPost, "/v1/compute/instances", nil, request, &res)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	if len(res.Data) != 1 {
		return MultipleDataObjectsError(nil)
	}
	instanceId := res.Data[0].InstanceId
	// Save the id before waiting: if waiting fails, the paid instance stays
	// in state (as tainted) instead of being ordered again on the next apply.
	d.SetId(strconv.FormatInt(instanceId, 10))

	_, diags := waitForInstanceReady(ctx, client, instanceId, d.Timeout(schema.TimeoutCreate))
	if diags.HasError() {
		return diags
	}
	diags = append(diags, duplicateInstanceWarnings(ctx, client, instanceId, d.Get("display_name").(string), startedAt)...)
	return append(diags, resourceInstanceRead(ctx, d, m)...)
}

// adoptInstance manages an existing instance without reinstalling it.
func adoptInstance(ctx context.Context, d *schema.ResourceData, m interface{}, existingId string) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(existingId, 10, 64)
	if err != nil {
		return diag.Errorf("existing_instance_id must be a numeric instance id: %s", err)
	}
	instance, httpResp, err := retrieveInstance(ctx, client, instanceId)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	d.SetId(existingId)

	if displayName := d.Get("display_name").(string); displayName != "" && displayName != instance.DisplayName {
		if diags := patchInstanceDisplayName(ctx, client, instanceId, displayName); diags.HasError() {
			return diags
		}
	}
	if diags := upgradeInstanceAddOns(ctx, client, instanceId, d.Get("add_ons").([]interface{})); diags.HasError() {
		return diags
	}
	return resourceInstanceRead(ctx, d, m)
}

func resourceInstanceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)

	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.Errorf("invalid instance id %q: %s", d.Id(), err)
	}

	instance, httpResp, err := retrieveInstance(ctx, client, instanceId)
	if err != nil {
		if removeFromStateIfNotFound(d, httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}

	values := flattenInstance(*instance)
	values["id"] = d.Id()
	values["add_ons"] = filterAddOnsToPrior(instance.AddOns, d.Get("add_ons").([]interface{}))
	if instance.ApplicationId != nil && *instance.ApplicationId != "" {
		values["application_id"] = *instance.ApplicationId
	}
	return setSchemaValues(d, resourceInstanceSchema(), values)
}

func resourceInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.Errorf("invalid instance id %q: %s", d.Id(), err)
	}

	var diags diag.Diagnostics

	if d.HasChange("display_name") {
		if displayName := d.Get("display_name").(string); displayName != "" {
			if diags := patchInstanceDisplayName(ctx, client, instanceId, displayName); diags.HasError() {
				return diags
			}
		}
	}

	if shouldReinstall(d) {
		if diags := reinstallInstance(ctx, client, instanceId, reinstallRequestFromResource(d), d.Timeout(schema.TimeoutUpdate)); diags.HasError() {
			return diags
		}
	}

	if d.HasChange("add_ons") {
		oldRaw, newRaw := d.GetChange("add_ons")
		if removed := addOnIdsOnlyIn(oldRaw.([]interface{}), newRaw.([]interface{})); len(removed) > 0 {
			diags = append(diags, warningDiag(
				"Add-ons are not removed",
				fmt.Sprintf("The API cannot remove add-ons from an instance. Add-ons %v stay on instance %d; cancel them in the customer control panel.", removed, instanceId),
			))
		}
		if upgradeDiags := upgradeInstanceAddOns(ctx, client, instanceId, newRaw.([]interface{})); upgradeDiags.HasError() {
			return append(diags, upgradeDiags...)
		}
	}

	_ = d.Set("last_updated", time.Now().Format(time.RFC850))
	return append(diags, resourceInstanceRead(ctx, d, m)...)
}

// shouldReinstall reports whether an attribute that requires a reinstall has
// changed to a value. Removing user_data or root_password from the
// configuration does not reinstall the instance.
func shouldReinstall(d *schema.ResourceData) bool {
	for _, attribute := range instanceReinstallAttributes {
		if !d.HasChange(attribute) {
			continue
		}
		switch value := d.Get(attribute).(type) {
		case string:
			if value != "" {
				return true
			}
		case int:
			if value > 0 {
				return true
			}
		case []interface{}:
			if len(value) > 0 {
				return true
			}
		}
	}
	return false
}

// reinstallRequestFromResource builds a reinstall request from every
// reinstall attribute, changed or not: the reinstall replaces the whole
// installation, so an attribute left out would be lost (issue #49: user_data
// was dropped when only image_id changed; issue #45: default_user was reset
// to admin).
func reinstallRequestFromResource(d *schema.ResourceData) map[string]interface{} {
	request := map[string]interface{}{
		"imageId": d.Get("image_id").(string),
	}
	setIfNotEmpty(request, "userData", d.Get("user_data").(string))
	setIfNotEmpty(request, "defaultUser", d.Get("default_user").(string))
	setIfNotEmpty(request, "applicationId", d.Get("application_id").(string))
	if sshKeys := interfacesToInt64s(d.Get("ssh_keys").([]interface{})); len(sshKeys) > 0 {
		request["sshKeys"] = sshKeys
	}
	if rootPassword := d.Get("root_password").(int); rootPassword > 0 {
		request["rootPassword"] = rootPassword
	}
	return request
}

func reinstallInstance(
	ctx context.Context,
	client *openapi.APIClient,
	instanceId int64,
	request map[string]interface{},
	timeout time.Duration,
) diag.Diagnostics {
	if imageId, _ := request["imageId"].(string); imageId == "" {
		instance, httpResp, err := retrieveInstance(ctx, client, instanceId)
		if err != nil {
			return apiErrorDiags(httpResp, err)
		}
		if instance.ImageId == nil || *instance.ImageId == "" {
			return diag.Errorf("cannot reinstall instance %d: image_id is not set and the instance reports no image", instanceId)
		}
		request["imageId"] = *instance.ImageId
	}
	httpResp, err := apiRequest(ctx, client, http.MethodPut, instancePath(instanceId), nil, request, nil)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	_, diags := waitForInstanceReady(ctx, client, instanceId, timeout)
	return diags
}

func patchInstanceDisplayName(ctx context.Context, client *openapi.APIClient, instanceId int64, displayName string) diag.Diagnostics {
	httpResp, err := apiRequest(ctx, client, http.MethodPatch, instancePath(instanceId), nil,
		map[string]interface{}{"displayName": displayName}, nil)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	return nil
}

// upgradeInstanceAddOns orders the configured add-ons that the instance does
// not have yet, as far as the upgrade API offers them.
func upgradeInstanceAddOns(ctx context.Context, client *openapi.APIClient, instanceId int64, configured []interface{}) diag.Diagnostics {
	wanted := addOnIds(configured)
	if len(wanted) == 0 {
		return nil
	}
	instance, httpResp, err := retrieveInstance(ctx, client, instanceId)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	missing := missingAddOnIds(wanted, instance.AddOns)
	if len(missing) == 0 {
		return nil
	}
	request := map[string]interface{}{}
	var unsupported []int64
	for _, id := range missing {
		if key, ok := instanceUpgradeAddOns[id]; ok {
			request[key] = map[string]interface{}{}
		} else {
			unsupported = append(unsupported, id)
		}
	}
	if len(unsupported) > 0 {
		return diag.Errorf("instance %d does not have add-ons %v and the API can only add private networking (1477) and backup (2214) to an existing instance. Order the add-ons in the customer control panel, or remove them from add_ons.", instanceId, unsupported)
	}
	httpResp, err = apiRequest(ctx, client, http.MethodPost, instancePath(instanceId)+"/upgrade", nil, request, nil)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	return nil
}

func resourceInstanceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.Errorf("invalid instance id %q: %s", d.Id(), err)
	}

	if !d.Get("cancel_on_destroy").(bool) {
		return diag.Diagnostics{warningDiag(
			"Instance not cancelled",
			fmt.Sprintf("cancel_on_destroy is false, so instance %d was only removed from the Terraform state. It keeps running and billing.", instanceId),
		)}
	}

	instance, httpResp, err := retrieveInstance(ctx, client, instanceId)
	if err != nil {
		if isNotFound(httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}
	if instance.cancelDate() != "" {
		return nil
	}

	httpResp, err = apiRequest(ctx, client, http.MethodPost, instancePath(instanceId)+"/cancel", nil, map[string]interface{}{}, nil)
	if err != nil {
		if isNotFound(httpResp) {
			return nil
		}
		diags := apiErrorDiags(httpResp, err)
		if httpResp != nil && httpResp.StatusCode >= 400 && httpResp.StatusCode < 500 {
			diags[0].Detail += "\n\nThe API refused to cancel the instance. Instances ordered in the customer control panel, or with a contract that cannot be cancelled yet, may not be cancellable through the API. Set cancel_on_destroy = false to remove the instance from the Terraform state without cancelling it."
		}
		return diags
	}
	return nil
}

// resourceInstanceCustomizeDiff rejects changes the API cannot make, at plan
// time instead of silently ignoring them during apply.
func resourceInstanceCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	if d.Id() == "" {
		return nil
	}
	for _, attribute := range []string{"product_id", "region"} {
		if !d.HasChange(attribute) {
			continue
		}
		oldValue, newValue := d.GetChange(attribute)
		if oldValue.(string) == "" || newValue.(string) == "" {
			continue
		}
		return fmt.Errorf(
			"%s cannot be changed from %q to %q: the Contabo API cannot change it on an existing instance, and replacing the instance would cancel it. Change it in the customer control panel and then update the configuration, or create a new instance",
			attribute, oldValue, newValue,
		)
	}

	if d.HasChange("add_ons") && m != nil {
		_, newRaw := d.GetChange("add_ons")
		wanted := addOnIds(newRaw.([]interface{}))
		var unsupported []int64
		for _, id := range wanted {
			if _, ok := instanceUpgradeAddOns[id]; !ok {
				unsupported = append(unsupported, id)
			}
		}
		if len(unsupported) == 0 {
			return nil
		}
		client, ok := m.(*openapi.APIClient)
		if !ok {
			return nil
		}
		instanceId, err := strconv.ParseInt(d.Id(), 10, 64)
		if err != nil {
			return nil
		}
		instance, _, err := retrieveInstance(ctx, client, instanceId)
		if err != nil {
			// Read reports API errors; do not fail the plan here.
			return nil
		}
		if missing := missingAddOnIds(unsupported, instance.AddOns); len(missing) > 0 {
			return fmt.Errorf(
				"add-ons %v cannot be added to existing instance %d: the API can only add private networking (1477) and backup (2214). Order them in the customer control panel or remove them from add_ons",
				missing, instanceId,
			)
		}
	}
	return nil
}

// duplicateInstanceWarnings warns about other instances with the same display
// name that were created during this create call. Issue #36 reports several
// instances ordered by one apply; the provider sends a single order, so this
// only makes such a case visible.
func duplicateInstanceWarnings(ctx context.Context, client *openapi.APIClient, instanceId int64, displayName string, startedAt time.Time) diag.Diagnostics {
	if displayName == "" {
		return nil
	}
	instances, _, err := listInstances(ctx, client, url.Values{"displayName": []string{displayName}})
	if err != nil {
		return nil
	}
	var duplicates []string
	for _, instance := range instances {
		if instance.InstanceId == instanceId || instance.DisplayName != displayName {
			continue
		}
		if instance.CreatedDate.After(startedAt.Add(-time.Minute)) {
			duplicates = append(duplicates, strconv.FormatInt(instance.InstanceId, 10))
		}
	}
	if len(duplicates) == 0 {
		return nil
	}
	return diag.Diagnostics{warningDiag(
		"Other instances with the same display name were just created",
		fmt.Sprintf("While instance %d was created, instances %s with display name %q were created too. If they are unexpected, cancel them to avoid charges.", instanceId, strings.Join(duplicates, ", "), displayName),
	)}
}

func setIfNotEmpty(request map[string]interface{}, key string, value string) {
	if value != "" {
		request[key] = value
	}
}

func expandInstanceAddOns(configured []interface{}) []map[string]interface{} {
	var out []map[string]interface{}
	for _, item := range configured {
		addOn, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, err := strconv.ParseInt(mapString(addOn, "id"), 10, 64)
		if err != nil || id == 0 {
			continue
		}
		quantity, _ := addOn["quantity"].(int)
		if quantity == 0 {
			quantity = 1
		}
		out = append(out, map[string]interface{}{"id": id, "quantity": quantity})
	}
	return out
}

func addOnIds(configured []interface{}) []int64 {
	var ids []int64
	for _, addOn := range expandInstanceAddOns(configured) {
		ids = append(ids, addOn["id"].(int64))
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func missingAddOnIds(wanted []int64, present []instanceAddOn) []int64 {
	have := map[int64]bool{}
	for _, addOn := range present {
		have[addOn.Id] = true
	}
	var missing []int64
	for _, id := range wanted {
		if !have[id] {
			missing = append(missing, id)
		}
	}
	return missing
}

// addOnIdsOnlyIn returns the ids in a that are not in b.
func addOnIdsOnlyIn(a []interface{}, b []interface{}) []int64 {
	inB := map[int64]bool{}
	for _, id := range addOnIds(b) {
		inB[id] = true
	}
	var out []int64
	for _, id := range addOnIds(a) {
		if !inB[id] {
			out = append(out, id)
		}
	}
	return out
}

// filterAddOnsToPrior keeps the add-ons listed in the prior state, so that
// add-ons the configuration does not mention (the API lists internal ones
// too) do not show up as changes. Without prior add-ons, for example after
// an import, it returns all of them.
func filterAddOnsToPrior(addOns []instanceAddOn, prior []interface{}) []interface{} {
	if len(addOnIds(prior)) == 0 {
		return flattenAddOns(addOns)
	}
	byId := map[int64]instanceAddOn{}
	for _, addOn := range addOns {
		byId[addOn.Id] = addOn
	}
	// Keep the order of the prior state, which follows the configuration.
	var kept []instanceAddOn
	for _, item := range prior {
		addOn, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, err := strconv.ParseInt(mapString(addOn, "id"), 10, 64)
		if err != nil {
			continue
		}
		if present, ok := byId[id]; ok {
			kept = append(kept, present)
			delete(byId, id)
		}
	}
	return flattenAddOns(kept)
}
