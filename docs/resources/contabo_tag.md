---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "contabo_tag Resource - terraform-provider-contabo-sdkv2"
subcategory: ""
description: |-
  Tags are Customer-defined labels which can be attached to any resource in your account. Tag API represent simple CRUD functions and allow you to manage your tags. Use tags to group your resources. For example you can define some user group with tag and give them permission to create compute instances.
---

# contabo_tag (Resource)

Tags are Customer-defined labels which can be attached to any resource in your account. Tag API represent simple CRUD functions and allow you to manage your tags. Use tags to group your resources. For example you can define some user group with tag and give them permission to create compute instances.

## Example Usage

```terraform
# Configure your Contabo API credentials
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Create a new tag
resource "contabo_tag" "default_tag" {
  color = "#000002"
  name="NewTag"
}

# Update an existing tag
resource "contabo_tag" "default_tag" {
	color 		  = "#ffffff"
  name = "UpdatedTag"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `color` (String) The tag color.
- `name` (String) The tag name.

### Read-Only

- `id` (String) The identifier of the tag. Use it to manage it!