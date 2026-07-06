package globalrule_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

// The count is deliberately huge: global rules apply to every request on the
// shared test stack, so the rule must never actually reject anything.
const testAccGlobalRuleBasic = `
resource "apisix_global_rule" "basic" {
  id = "tf-acc-global-rule-basic"

  plugins = {
    "limit-count" = jsonencode({
      count         = 100000
      time_window   = 60
      rejected_code = 429
    })
  }
}
`

const testAccGlobalRuleUpdated = `
resource "apisix_global_rule" "basic" {
  id = "tf-acc-global-rule-basic"

  plugins = {
    "limit-count" = jsonencode({
      count         = 200000
      time_window   = 60
      rejected_code = 429
    })
  }
}
`

func TestAccGlobalRule_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGlobalRuleBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_global_rule.basic", "id", "tf-acc-global-rule-basic"),
				),
			},
			{
				Config: testAccGlobalRuleUpdated,
			},
			{
				Config:             testAccGlobalRuleUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_global_rule.basic",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
