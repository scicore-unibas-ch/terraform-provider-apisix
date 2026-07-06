package route_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

// Covers matching attributes (uri/hosts/methods/vars), plugins, and an inline
// upstream — the shared codec path through apisix_route.
const testAccRouteBasic = `
resource "apisix_route" "basic" {
  id      = "tf-acc-route-basic"
  name    = "tf-acc-route"
  desc    = "state-stability smoke test"
  uri     = "/tf-acc/*"
  hosts   = ["tf-acc.example.com"]
  methods = ["GET", "POST"]

  vars = jsonencode([
    ["arg_debug", "==", "1"],
  ])

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
    timeout = {
      connect = 3
      send    = 3
      read    = 6
    }
  }

  labels = {
    managed_by = "terraform"
  }
}
`

const testAccRouteUpdated = `
resource "apisix_route" "basic" {
  id       = "tf-acc-route-basic"
  name     = "tf-acc-route"
  desc     = "updated description"
  uri      = "/tf-acc/*"
  hosts    = ["tf-acc.example.com", "tf-acc-2.example.com"]
  methods  = ["GET"]
  priority = 5

  vars = jsonencode([
    ["arg_debug", "==", "1"],
  ])

  plugins = {
    "limit-count" = jsonencode({
      count         = 20000
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
    timeout = {
      connect = 3
      send    = 3
      read    = 6
    }
  }

  labels = {
    managed_by = "terraform"
  }
}
`

// The shared upstream validators must also fire on the inline block (nested
// paths resolve relative to the upstream object).
func TestAccRoute_inlineUpstreamValidators(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "apisix_route" "rewrite_no_host" {
  id  = "tf-acc-route-rewrite-no-host"
  uri = "/tf-acc-invalid/*"
  upstream = {
    nodes     = [{ host = "127.0.0.1", port = 8081 }]
    pass_host = "rewrite"
  }
}
`,
				ExpectError: regexp.MustCompile(`Missing upstream_host`),
			},
		},
	})
}

func TestAccRoute_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRouteBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_route.basic", "id", "tf-acc-route-basic"),
					// Schema defaults must be materialized in state.
					resource.TestCheckResourceAttr("apisix_route.basic", "status", "1"),
					resource.TestCheckResourceAttr("apisix_route.basic", "priority", "0"),
					resource.TestCheckResourceAttr("apisix_route.basic", "enable_websocket", "false"),
					// Inline upstream defaults through the shared codec.
					resource.TestCheckResourceAttr("apisix_route.basic", "upstream.type", "roundrobin"),
					resource.TestCheckResourceAttr("apisix_route.basic", "upstream.nodes.0.weight", "1"),
				),
			},
			{
				Config: testAccRouteUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_route.basic", "desc", "updated description"),
					resource.TestCheckResourceAttr("apisix_route.basic", "priority", "5"),
					resource.TestCheckResourceAttr("apisix_route.basic", "hosts.#", "2"),
				),
			},
			{
				Config:             testAccRouteUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_route.basic",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
