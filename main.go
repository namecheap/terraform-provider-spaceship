package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"terraform-provider-spaceship/internal/provider"
)

var (
	version = "dev"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/namecheap/spaceship",
	})
	if err != nil {
		log.Fatal(err)
	}
}
