package contabo

import (
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// flattenInstance returns the Terraform values of an instance, keyed by
// attribute name. Resources and data sources set the keys their schema has.
func flattenInstance(instance instanceResponse) map[string]interface{} {
	values := map[string]interface{}{
		"name":              instance.Name,
		"display_name":      instance.DisplayName,
		"region":            instance.Region,
		"region_name":       instance.RegionName,
		"data_center":       instance.DataCenter,
		"product_id":        instance.ProductId,
		"product_name":      instance.ProductName,
		"product_type":      instance.ProductType,
		"ip_config":         flattenInstanceIpConfig(instance),
		"additional_ips":    flattenAdditionalIps(instance.AdditionalIps),
		"additional_ips_v4": flattenAdditionalIpsV4(instance.AdditionalIps),
		"mac_address":       instance.MacAddress,
		"ram_mb":            int(instance.RamMb),
		"cpu_cores":         int(instance.CpuCores),
		"disk_mb":           int(instance.DiskMb),
		"os_type":           instance.OsType,
		"ssh_keys":          int64sToInterfaces(instance.SshKeys),
		"created_date":      instance.CreatedDate.Format(time.RFC850),
		"cancel_date":       instance.cancelDate(),
		"status":            instance.Status,
		"v_host_id":         int(instance.VHostId),
		"add_ons":           flattenAddOns(instance.AddOns),
		"error_message":     instance.errorMessage(),
	}
	if instance.ImageId != nil {
		values["image_id"] = *instance.ImageId
	}
	if instance.DefaultUser != nil {
		values["default_user"] = *instance.DefaultUser
	}
	return values
}

// setSchemaValues sets every value whose key is an attribute of the
// resource schema.
func setSchemaValues(d *schema.ResourceData, resourceSchema map[string]*schema.Schema, values map[string]interface{}) diag.Diagnostics {
	for key, value := range values {
		if _, ok := resourceSchema[key]; !ok {
			continue
		}
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("could not set %s: %s", key, err)
		}
	}
	return nil
}

func flattenInstanceIpConfig(instance instanceResponse) []interface{} {
	return []interface{}{map[string]interface{}{
		"v4": []interface{}{flattenInstanceIp(instance.IpConfig.V4)},
		"v6": []interface{}{flattenInstanceIp(instance.IpConfig.V6)},
	}}
}

func flattenInstanceIp(ip instanceIp) map[string]interface{} {
	return map[string]interface{}{
		"ip":           ip.Ip,
		"netmask_cidr": int(ip.NetmaskCidr),
		"gateway":      ip.Gateway,
	}
}

func flattenAdditionalIps(ips []instanceIpWrapper) []interface{} {
	out := make([]interface{}, 0, len(ips))
	for _, ip := range ips {
		out = append(out, map[string]interface{}{
			"v4": []interface{}{flattenInstanceIp(ip.V4)},
		})
	}
	return out
}

func flattenAdditionalIpsV4(ips []instanceIpWrapper) []interface{} {
	out := make([]interface{}, 0, len(ips))
	for _, ip := range ips {
		out = append(out, flattenInstanceIp(ip.V4))
	}
	return out
}

func flattenAddOns(addOns []instanceAddOn) []interface{} {
	out := make([]interface{}, 0, len(addOns))
	for _, addOn := range addOns {
		out = append(out, map[string]interface{}{
			"id":       strconv.FormatInt(addOn.Id, 10),
			"quantity": int(addOn.Quantity),
		})
	}
	return out
}

func int64sToInterfaces(values []int64) []interface{} {
	out := make([]interface{}, 0, len(values))
	for _, value := range values {
		out = append(out, int(value))
	}
	return out
}

func interfacesToInt64s(values []interface{}) []int64 {
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if i, ok := value.(int); ok {
			out = append(out, int64(i))
		}
	}
	return out
}
