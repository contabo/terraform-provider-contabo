---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "contabo_object_storage Data Source - terraform-provider-contabo-sdkv2"
subcategory: ""
description: |-
  
---

# contabo_object_storage (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `auto_scaling` (Block List) (see [below for nested schema](#nestedblock--auto_scaling))

### Read-Only

- `cancel_date` (String)
- `created_date` (String)
- `customer_id` (String)
- `data_center` (String)
- `id` (String) The ID of this resource.
- `region` (String)
- `s3_tenant_id` (String)
- `s3_url` (String)
- `status` (String)
- `tenant_id` (String)
- `total_purchased_space_tb` (Number)

<a id="nestedblock--auto_scaling"></a>
### Nested Schema for `auto_scaling`

Optional:

- `error_message` (String)
- `size_limit_tb` (Number)
- `state` (String)

