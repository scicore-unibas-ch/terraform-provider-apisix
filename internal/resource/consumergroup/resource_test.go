package consumergroup_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/provider"
)

// resource.Test auto-skips unless TF_ACC=1. It still requires a live APISIX —
// `make test-env-up` brings up the docker-compose stack used by the bash
// acceptance scripts; the same env vars (APISIX_BASE_URL, APISIX_ADMIN_KEY)
// configure the provider here.
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

// TestAccConsumerGroup_stateStability is a minimal skeleton covering the three
// guarantees that bash acceptance scripts cannot easily verify:
//
//  1. Create succeeds and the resource is registered in state.
//  2. A no-op re-plan against the same config produces an empty plan
//     (i.e. Read + plan modifiers reconcile any server-side normalization).
//  3. Import round-trips: importing by ID reconstructs the same state.
func TestAccConsumerGroup_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories,
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
				Config:             testAccConsumerGroupBasic,
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
