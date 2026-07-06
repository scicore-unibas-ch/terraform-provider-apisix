// Package acctest holds the boilerplate shared by the per-resource Go
// acceptance tests (resource.Test auto-skips unless TF_ACC=1; the tests need
// the live APISIX docker-compose stack from `make test-env-up`, configured
// through the same APISIX_BASE_URL / APISIX_ADMIN_KEY env vars as the bash
// acceptance scripts).
package acctest

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/provider"
)

// ProtoV6ProviderFactories serves the provider in-process for resource.Test.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"apisix": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// PreCheck fails fast when the required environment is missing.
func PreCheck(t *testing.T) {
	t.Helper()
	for _, k := range []string{"APISIX_BASE_URL", "APISIX_ADMIN_KEY"} {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for TF_ACC tests", k)
		}
	}
}
