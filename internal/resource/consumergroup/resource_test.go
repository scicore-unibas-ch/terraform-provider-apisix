package consumergroup_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

const testAccConsumerGroupBasic = `
resource "apisix_consumer_group" "basic" {
  id   = "tf-acc-consumer-group-basic"
  desc = "state-stability smoke test"

  plugins = {
    "limit-count" = jsonencode({
      count         = 10000
      time_window   = 60
      rejected_code = 429
    })
  }
}
`

const testAccConsumerGroupUpdated = `
resource "apisix_consumer_group" "basic" {
  id   = "tf-acc-consumer-group-basic"
  desc = "updated description"

  plugins = {
    "limit-count" = jsonencode({
      count         = 20000
      time_window   = 60
      rejected_code = 429
    })
  }

  labels = {
    managed_by = "terraform"
  }
}
`

// TestAccConsumerGroup_stateStability covers the guarantees that bash
// acceptance scripts cannot easily verify:
//
//  1. Create succeeds and the resource is registered in state.
//  2. Update in place (PUT full replace) converges.
//  3. A no-op re-plan against the same config produces an empty plan
//     (i.e. Read + plan modifiers reconcile any server-side normalization).
//  4. Import round-trips: importing by ID reconstructs the same state.
func TestAccConsumerGroup_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConsumerGroupBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"apisix_consumer_group.basic", "id", "tf-acc-consumer-group-basic",
					),
					resource.TestCheckResourceAttr(
						"apisix_consumer_group.basic", "desc", "state-stability smoke test",
					),
				),
			},
			{
				Config: testAccConsumerGroupUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"apisix_consumer_group.basic", "desc", "updated description",
					),
				),
			},
			{
				Config:             testAccConsumerGroupUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_consumer_group.basic",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
