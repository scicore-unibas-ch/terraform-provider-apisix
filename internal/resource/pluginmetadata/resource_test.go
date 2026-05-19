package pluginmetadata_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/provider"
)

var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"apisix": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, k := range []string{"APISIX_BASE_URL", "APISIX_ADMIN_KEY"} {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for TF_ACC tests", k)
		}
	}
}

const testAccPluginMetadataSyslog = `
resource "apisix_plugin_metadata" "syslog" {
  id = "syslog"
  metadata = jsonencode({
    log_format = {
      host       = "$host"
      "@timestamp" = "$time_iso8601"
      client_ip  = "$remote_addr"
    }
  })
}
`

// Same logical metadata as above with keys in a different order. With the
// jsonstr.SuppressEquivalent plan modifier this must produce an empty plan.
const testAccPluginMetadataSyslogReordered = `
resource "apisix_plugin_metadata" "syslog" {
  id = "syslog"
  metadata = jsonencode({
    log_format = {
      "@timestamp" = "$time_iso8601"
      client_ip  = "$remote_addr"
      host       = "$host"
    }
  })
}
`

func TestAccPluginMetadata_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluginMetadataSyslog,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_plugin_metadata.syslog", "id", "syslog"),
				),
			},
			{
				Config:             testAccPluginMetadataSyslog,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config:             testAccPluginMetadataSyslogReordered,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_plugin_metadata.syslog",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
