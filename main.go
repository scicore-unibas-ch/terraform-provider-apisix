package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/provider"
)

var (
	version = "dev"
	commit  = "unknown"
)

var _ = commit // silence unused warning; commit is set via -ldflags at release time

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run with debugger support")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.opentofu.org/scicore-unibas-ch/apisix",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
