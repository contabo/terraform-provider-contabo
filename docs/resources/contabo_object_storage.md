---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "contabo_object_storage Resource - terraform-provider-contabo-sdkv2"
subcategory: ""
description: |-
  Manage S3 compatible Object Storage. With the Object Storage API you can create Object Storages in different locations. Please note that you can only have one Object Storage per location. Furthermore, you can increase the amount of storage space and control the autoscaling feature which allows you to automatically perform a monthly upgrade of the disk space to the specified maximum. You might also inspect the usage. This API is not the S3 API itself. For accessing the S3 API directly or with S3 compatible tools like aws cli and after having created / upgraded your Object Storage please use the S3 URL from this Storage API and refer to the User Mangement API to retrieve the S3 credentials.
---

# contabo_object_storage (Resource)

Manage S3 compatible Object Storage. With the Object Storage API you can create Object Storages in different locations. Please note that you can only have one Object Storage per location. Furthermore, you can increase the amount of storage space and control the autoscaling feature which allows you to automatically perform a monthly upgrade of the disk space to the specified maximum. You might also inspect the usage. This API is not the S3 API itself. For accessing the S3 API directly or with S3 compatible tools like `aws` cli and after having created / upgraded your Object Storage please use the S3 URL from this Storage API and refer to the User Mangement API to retrieve the S3 credentials.

## Example Usage

```terraform
# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Create a new object storage in region EU
resource "contabo_object_storage" "object_storage_eu" {
  region                   = "EU"
	total_purchased_space_tb = 2
}

# Update a new object storage, enable autoscaling
resource "contabo_object_storage" "object_storage_eu" {
  auto_scaling {
    state         = "enabled"
    size_limit_tb = 5
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `region` (String) Region where the Object Storage should be located. Default region is the EU. Following regions are available: `EU`,`US-central`, `SIN`.
- `total_purchased_space_tb` (Number) Amount of purchased / requested object storage in terabyte.

### Optional

- `auto_scaling` (Block List) (see [below for nested schema](#nestedblock--auto_scaling))
- `display_name` (String) Display name for object storage.

### Read-Only

- `cancel_date` (String) The date on which the Object Storage will be cancelled and therefore no longer available.
- `created_date` (String) The creation date of the Object Storage.
- `customer_id` (String) Your customer number.
- `data_center` (String) Data center the object storage is located in.
- `id` (String) The identifier of the Object Storage. Use it to manage it!
- `s3_tenant_id` (String) Your S3 tenant Id. Only required for public sharing.
- `s3_url` (String) S3 URL to connect to your S3 compatible Object Storage.
- `status` (String) The object storage status. It can be set to `PROVISIONING`,`READY`,`UPGRADING`,`CANCELLED`,`ERROR` or `DISABLED`.
- `tenant_id` (String) Your customer tenant Id.

<a id="nestedblock--auto_scaling"></a>
### Nested Schema for `auto_scaling`

Optional:

- `error_message` (String) If the autoscaling is in an error state (see status property), the error message can be seen in this field.
- `size_limit_tb` (Number) Autoscaling size limit for the current object storage.
- `state` (String) Status of this object storage.  It can be set to `enabled`, `disabled` or `error`.
