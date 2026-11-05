package contabo

import (
	"context"
	"strconv"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// instanceComputedAttributes returns the read-only instance attributes shared
// by the contabo_instance and contabo_instances data sources.
func instanceComputedAttributes() map[string]*schema.Schema {
	computedString := func(description string) *schema.Schema {
		return &schema.Schema{Type: schema.TypeString, Computed: true, Description: description}
	}
	computedInt := func(description string) *schema.Schema {
		return &schema.Schema{Type: schema.TypeInt, Computed: true, Description: description}
	}
	return map[string]*schema.Schema{
		"name":          computedString("The name that Contabo gave the instance, for example `vmi123456`."),
		"display_name":  computedString("The display name of the instance. The Customer Control Panel shows this name."),
		"image_id":      computedString("The id of the image that the instance was installed with."),
		"region":        computedString("The region of the instance, for example `EU`."),
		"region_name":   computedString("The full name of the region of the instance, for example `European Union`."),
		"data_center":   computedString("The data center of the instance, for example `European Union 2`."),
		"product_id":    computedString("The product id of the instance, for example `V153`."),
		"product_name":  computedString("The name of the product of the instance, for example `Cloud VPS 4`."),
		"product_type":  computedString("The category of the product of the instance, for example `ssd`, `nvme` or `vds`."),
		"ip_config":     instanceIpConfigSchema(),
		"mac_address":   computedString("The MAC address of the instance."),
		"ram_mb":        computedInt("The size of the memory (RAM) of the instance in megabytes."),
		"cpu_cores":     computedInt("The number of CPU cores of the instance."),
		"disk_mb":       computedInt("The size of the disk of the instance in megabytes."),
		"os_type":       computedString("The type of operating system that is installed on the instance, for example `Linux`."),
		"created_date":  computedString("The date when the instance was created."),
		"cancel_date":   computedString("The date when the instance will be cancelled. The value is empty when the instance is not cancelled."),
		"status":        computedString("The status of the instance, for example `running`, `stopped` or `installing`."),
		"v_host_id":     computedInt("The id of the host system that runs the instance."),
		"error_message": computedString("The error message of the instance. The value is only set when the status of the instance is `error`."),
		"default_user":  computedString("The default user that the installation created, for example `root` or `admin`."),
		"ssh_keys": {
			Type:        schema.TypeList,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeInt},
			Description: "The secret ids of the SSH keys that the installation added for the default user.",
		},
		"add_ons": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "The list of all add-ons of the instance.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id":       computedString("The id of the add-on."),
					"quantity": computedInt("The number of units of the add-on."),
				},
			},
		},
		"additional_ips": instanceAdditionalIpsSchema(),
	}
}

func dataSourceInstance() *schema.Resource {
	attributes := instanceComputedAttributes()
	attributes["id"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The id of the instance that the data source reads.",
	}
	// Kept for compatibility with earlier versions of this data source. The
	// API does not return these values, so they are always empty.
	for name, description := range map[string]string{
		"last_updated": "This attribute is deprecated, and the value is always empty.",
		"user_data":    "This attribute is deprecated. The API does not return the user data, so the value is always empty.",
		"license":      "This attribute is deprecated. The API does not return the license, so the value is always empty.",
	} {
		attributes[name] = &schema.Schema{Type: schema.TypeString, Computed: true, Description: description}
	}
	attributes["period"] = &schema.Schema{Type: schema.TypeInt, Computed: true, Description: "This attribute is deprecated. The API does not return the contract period, so the value is always 0."}
	attributes["additional_ips_v4"] = &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The additional IPv4 addresses of the instance. This attribute has the same data as `additional_ips`.",
		Elem:        instanceIpSchema(),
	}

	return &schema.Resource{
		Description: "This data source reads one compute instance by the id of the instance. To search for instances, use the `contabo_instances` data source.",
		ReadContext: dataSourceInstanceRead,
		Schema:      attributes,
	}
}

func dataSourceInstanceRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	client := m.(*openapi.APIClient)

	instanceId, err := strconv.ParseInt(d.Get("id").(string), 10, 64)
	if err != nil {
		return diag.Errorf("id must be a numeric instance id: %s", err)
	}

	instance, httpResp, err := retrieveInstance(ctx, client, instanceId)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}

	d.SetId(strconv.FormatInt(instance.InstanceId, 10))
	return setSchemaValues(d, dataSourceInstance().Schema, flattenInstance(*instance))
}
