package main

import (
	"contabo.com/terraform-provider-contabo/contabo"

	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

// Generate the Terraform provider documentation using `tfplugindocs`.
// --provider-name must match the resource prefix ("contabo_*"); otherwise
// newer tfplugindocs derives it from the repo dir ("contabo-sdkv2") and fails
// schema lookup.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name contabo

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	plugin.Serve(&plugin.ServeOpts{
		Debug:        debug,
		ProviderAddr: "contabo/contabo",
		ProviderFunc: func() *schema.Provider {
			return contabo.Provider()
		},
	})
}
