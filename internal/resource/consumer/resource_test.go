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

// An Optional string the practitioner sets to "" must round-trip. The Admin
// API cannot tell "absent" from "empty", so a decode that collapses "" to null
// leaves config and state disagreeing for ever: the resource reports "will be
// updated in-place" on every plan, with no attribute rendered, and no amount of
// applying settles it.
//
// This is easy to reach without meaning to. A module writing
// `desc = var.description` where the variable is `optional(string, "")` sends
// "" for every entry that does not set one.
//
// The PlanOnly step is the assertion: before the fix it failed with a non-empty
// plan. The second one repeats it after an apply of the identical config, since
// the first plan is compared against state written from the plan, and only the
// refresh afterwards reintroduces the null.
const testAccConsumerEmptyDesc = `
resource "apisix_consumer" "empty_desc" {
  id   = "tf_acc_consumer_empty_desc"
  desc = ""

  plugins = {
    "key-auth" = jsonencode({
      key = "tf-acc-empty-desc-key"
    })
  }
}
`

func TestAccConsumer_emptyDescIsStable(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConsumerEmptyDesc,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_consumer.empty_desc", "desc", ""),
				),
			},
			{
				Config:             testAccConsumerEmptyDesc,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccConsumerEmptyDesc,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_consumer.empty_desc", "desc", ""),
				),
			},
			{
				Config:             testAccConsumerEmptyDesc,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

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
