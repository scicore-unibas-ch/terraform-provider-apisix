package service_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

const testAccServiceBasic = `
resource "apisix_service" "basic" {
  id   = "tf-acc-service-basic"
  name = "tf-acc-service"
  desc = "state-stability smoke test"

  hosts = ["tf-acc-svc.example.com"]

  plugins = {
    "limit-count" = jsonencode({
      count         = 10000
      time_window   = 60
      rejected_code = 429
    })
  }

  upstream = {
    nodes = [
      {
        host = "127.0.0.1"
        port = 8081
      },
    ]
  }

  labels = {
    managed_by = "terraform"
  }
}
`

const testAccServiceUpdated = `
resource "apisix_service" "basic" {
  id   = "tf-acc-service-basic"
  name = "tf-acc-service"
  desc = "updated description"

  hosts = ["tf-acc-svc.example.com"]

  enable_websocket = true

  plugins = {
    "limit-count" = jsonencode({
      count         = 10000
      time_window   = 60
      rejected_code = 429
    })
  }

  upstream = {
    nodes = [
      {
        host = "127.0.0.1"
        port = 8081
      },
    ]
  }

  labels = {
    managed_by = "terraform"
  }
}
`

func TestAccService_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_service.basic", "id", "tf-acc-service-basic"),
					resource.TestCheckResourceAttr("apisix_service.basic", "enable_websocket", "false"),
					resource.TestCheckResourceAttr("apisix_service.basic", "upstream.scheme", "http"),
					resource.TestCheckResourceAttr("apisix_service.basic", "upstream.nodes.0.weight", "1"),
				),
			},
			{
				Config: testAccServiceUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_service.basic", "desc", "updated description"),
					resource.TestCheckResourceAttr("apisix_service.basic", "enable_websocket", "true"),
				),
			},
			{
				Config:             testAccServiceUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_service.basic",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
