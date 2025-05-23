---
page_title: "contabo-terraform-provider"
subcategory: ""
description: |-
  A terraform provider for managing your products from Contabo like Cloud VPS and VDS.

---

# Contabo Provider

A terraform provider for managing resources offered by [Contabo](https://contabo.com) like Cloud VPS, VDS or S3 compatible Object Storage. For proper usage credentials are required.


## Example Usage

```terraform
terraform {
  required_providers {
    contabo = {
      source = "contabo/contabo"
      version = ">= 0.1.32"
    }
  }
}

# Configure your Contabo API credentials in provider stanza
provider "contabo" {
  oauth2_client_id     = "[your client id]"
  oauth2_client_secret = "[your client secret]"
  oauth2_user          = "[your username]"
  oauth2_pass          = "[your password]"
}

# Create a default contabo VPS instance
resource "contabo_instance" "default_instance" {}

# Output our newly created instances
output "default_instance_output" {
  description = "Our first default instance"
  value       = contabo_instance.default_instance
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `api` (String) The api endpoint is https://api.contabo.com.
- `oauth2_client_id` (String) Your oauth2 client id can be found in the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.
- `oauth2_client_secret` (String) Your oauth2 client secret can be found in the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.
- `oauth2_pass` (String) API Password (this is a new password which you'll set or change in the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.)
- `oauth2_token_url` (String) The oauth2 token url is https://auth.contabo.com/auth/realms/contabo/protocol/openid-connect/token.
- `oauth2_user` (String) API User (your email address to login to the [Customer Control Panel](https://new.contabo.com/account/security) under the menu item account secret.
