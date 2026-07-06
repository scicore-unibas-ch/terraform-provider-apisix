package pluginmetadata_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

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
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
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
