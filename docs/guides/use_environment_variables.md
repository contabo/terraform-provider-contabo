---
subcategory: ""
page_title: "Configure provider with environment variables"
description: |-
    An example configuring the contabo-terraform-provider via environment variables.
---

# Minimal provider configuration example

```terraform
terraform {
  required_providers {
    contabo = {
      source = "contabo/contabo"
      version = "v0.1.3"
    }
  }
}

# Set the following environment variables:
#
# CNTB_OAUTH2_CLIENT_ID
# CNTB_OAUTH2_CLIENT_SECRET
# CNTB_OAUTH2_USER
# CNTB_OAUTH2_PASS
#
# and you are good to go
provider "contabo" {}


# Create a default contabo VPS instance
resource "contabo_instance" "default_instance" {}

# Output our newly created instances
output "default_instance_output" {
  description = "Our first default instance"
  value = contabo_instance.default_instance
}
```
