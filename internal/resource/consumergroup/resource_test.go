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

const testAccConsumerGroupNoPlugins = `
resource "apisix_consumer_group" "empty" {
  id      = "tf-acc-consumer-group-empty"
  desc    = "carries no plugins"
  plugins = {}
}
`

// TestAccConsumerGroup_emptyPlugins covers a group that carries no plugin
// config at all.
//
// This is a real shape: a consumer group is also the unit of *attribution* —
// APISIX exposes it as $consumer_group_id, which is what per-group Prometheus
// labels and log lines are keyed on. A deployment that puts every quota on the
// consumer still needs the group objects, and has nothing to put in them.
//
// It regressed because apiBody.Plugins was tagged `json:",omitempty"`, which
// drops a zero-length map as well as a nil one: the field never reached the
// Admin API and create failed with `property "plugins" is required`. APISIX
// requires the key to be present, not to hold anything — `{"plugins":{}}` is
// accepted (verified against 3.14 through 3.18).
func TestAccConsumerGroup_emptyPlugins(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConsumerGroupNoPlugins,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"apisix_consumer_group.empty", "id", "tf-acc-consumer-group-empty",
					),
					resource.TestCheckResourceAttr(
						"apisix_consumer_group.empty", "plugins.%", "0",
					),
				),
			},
			// An empty map must survive Read: if the provider decoded "no
			// plugins" as null it would show a permanent diff.
			{
				Config:             testAccConsumerGroupNoPlugins,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_consumer_group.empty",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
