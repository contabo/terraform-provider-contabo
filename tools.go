//go:build tools
// +build tools

// This file tracks tooling dependencies (tfplugindocs, used via `go generate`
// in main.go) so `go mod tidy` keeps them in go.mod/go.sum. It is never
// compiled into the provider binary — the `tools` build tag excludes it.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
