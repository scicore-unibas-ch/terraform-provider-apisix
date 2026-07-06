package pluginconfig_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

const testAccPluginConfigBasic = `
resource "apisix_plugin_config" "basic" {
  id   = "tf-acc-plugin-config-basic"
  desc = "state-stability smoke test"

  plugins = {
    "response-rewrite" = jsonencode({
      headers = {
        set = {
          "X-Test" = "tf-acc"
        }
      }
    })
  }

  labels = {
    managed_by = "terraform"
  }
}
`

const testAccPluginConfigUpdated = `
resource "apisix_plugin_config" "basic" {
  id   = "tf-acc-plugin-config-basic"
  desc = "updated description"

  plugins = {
    "response-rewrite" = jsonencode({
      headers = {
        set = {
          "X-Test" = "tf-acc-v2"
        }
      }
    })
  }

  labels = {
    managed_by = "terraform"
  }
}
`

func TestAccPluginConfig_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluginConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_plugin_config.basic", "id", "tf-acc-plugin-config-basic"),
					resource.TestCheckResourceAttr("apisix_plugin_config.basic", "desc", "state-stability smoke test"),
				),
			},
			{
				Config: testAccPluginConfigUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_plugin_config.basic", "desc", "updated description"),
				),
			},
			{
				Config:             testAccPluginConfigUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_plugin_config.basic",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
