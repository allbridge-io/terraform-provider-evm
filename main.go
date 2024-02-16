// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"
	"terraform-provider-evm/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

//go:generate ./tools/generate-docs.sh

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// TODO: Update this string with the published name of your provider.
		Address: "registry.terraform.io/hashicorp/evm",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version, nil), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
