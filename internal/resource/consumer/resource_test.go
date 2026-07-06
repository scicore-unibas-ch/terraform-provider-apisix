package consumer_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

// APISIX consumer usernames only allow [a-zA-Z0-9_].
const testAccConsumerBasic = `
resource "apisix_consumer" "basic" {
  id   = "tf_acc_consumer_basic"
  desc = "state-stability smoke test"

  plugins = {
    "key-auth" = jsonencode({
      key = "tf-acc-secret-key"
    })
  }

  labels = {
    managed_by = "terraform"
  }
}
`

const testAccConsumerUpdated = `
resource "apisix_consumer" "basic" {
  id   = "tf_acc_consumer_basic"
  desc = "updated description"

  plugins = {
    "key-auth" = jsonencode({
      key = "tf-acc-rotated-key"
    })
  }

  labels = {
    managed_by = "terraform"
  }
}
`

func TestAccConsumer_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConsumerBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_consumer.basic", "id", "tf_acc_consumer_basic"),
					resource.TestCheckResourceAttr("apisix_consumer.basic", "desc", "state-stability smoke test"),
				),
			},
			{
				Config: testAccConsumerUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_consumer.basic", "desc", "updated description"),
				),
			},
			{
				Config:             testAccConsumerUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_consumer.basic",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
