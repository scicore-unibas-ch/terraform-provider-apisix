package upstream_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

// Exercises every nested block of the shared upstream codec: nodes (explicit
// and defaulted weight/priority), timeout, a partial keepalive_pool (inner
// defaults must fill), health_check JSON, labels, retries.
const testAccUpstreamFull = `
resource "apisix_upstream" "full" {
  id   = "tf-acc-upstream-full"
  name = "tf-acc-full"
  desc = "state-stability smoke test"
  type = "roundrobin"

  nodes = [
    {
      host = "127.0.0.1"
      port = 8081
    },
    {
      host     = "127.0.0.1"
      port     = 8082
      weight   = 2
      priority = 1
    },
  ]

  timeout = {
    connect = 3
    send    = 3
    read    = 6
  }

  keepalive_pool = {
    size = 64
  }

  health_check = jsonencode({
    active = {
      type      = "http"
      http_path = "/health"
      healthy = {
        interval  = 2
        successes = 1
      }
      unhealthy = {
        interval      = 1
        http_failures = 2
      }
    }
  })

  retries = 2

  labels = {
    managed_by = "terraform"
  }
}
`

const testAccUpstreamFullUpdated = `
resource "apisix_upstream" "full" {
  id   = "tf-acc-upstream-full"
  name = "tf-acc-full"
  desc = "updated description"
  type = "roundrobin"

  nodes = [
    {
      host   = "127.0.0.1"
      port   = 8081
      weight = 3
    },
  ]

  timeout = {
    connect = 5
    send    = 5
    read    = 10
  }

  keepalive_pool = {
    size = 64
  }

  health_check = jsonencode({
    active = {
      type      = "http"
      http_path = "/health"
      healthy = {
        interval  = 2
        successes = 1
      }
      unhealthy = {
        interval      = 1
        http_failures = 2
      }
    }
  })

  retries = 1

  labels = {
    managed_by = "terraform"
  }
}
`

// The cross-attribute rules shared with the inline upstream block must fail
// at plan time, before anything reaches APISIX.
func TestAccUpstream_configValidators(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "apisix_upstream" "conflict" {
  id             = "tf-acc-upstream-conflict"
  nodes          = [{ host = "127.0.0.1", port = 8081 }]
  service_name   = "api-service"
  discovery_type = "consul"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: `
resource "apisix_upstream" "orphan_key" {
  id    = "tf-acc-upstream-orphan-key"
  nodes = [{ host = "127.0.0.1", port = 8081 }]
  tls = {
    client_cert = "dummy"
  }
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: `
resource "apisix_upstream" "rewrite_no_host" {
  id        = "tf-acc-upstream-rewrite-no-host"
  nodes     = [{ host = "127.0.0.1", port = 8081 }]
  pass_host = "rewrite"
}
`,
				ExpectError: regexp.MustCompile(`Missing upstream_host`),
			},
			{
				// The valid rewrite combination must still apply cleanly.
				Config: `
resource "apisix_upstream" "rewrite_ok" {
  id            = "tf-acc-upstream-rewrite-ok"
  nodes         = [{ host = "127.0.0.1", port = 8081 }]
  pass_host     = "rewrite"
  upstream_host = "internal.example.com"
}
`,
				Check: resource.TestCheckResourceAttr(
					"apisix_upstream.rewrite_ok", "upstream_host", "internal.example.com",
				),
			},
		},
	})
}

func TestAccUpstream_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUpstreamFull,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_upstream.full", "id", "tf-acc-upstream-full"),
					// Schema defaults must be materialized in state.
					resource.TestCheckResourceAttr("apisix_upstream.full", "scheme", "http"),
					resource.TestCheckResourceAttr("apisix_upstream.full", "pass_host", "pass"),
					resource.TestCheckResourceAttr("apisix_upstream.full", "nodes.0.weight", "1"),
					resource.TestCheckResourceAttr("apisix_upstream.full", "nodes.1.weight", "2"),
					// Partial keepalive_pool: inner defaults fill in.
					resource.TestCheckResourceAttr("apisix_upstream.full", "keepalive_pool.size", "64"),
					resource.TestCheckResourceAttr("apisix_upstream.full", "keepalive_pool.idle_timeout", "60"),
					resource.TestCheckResourceAttr("apisix_upstream.full", "keepalive_pool.requests", "1000"),
				),
			},
			{
				Config: testAccUpstreamFullUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("apisix_upstream.full", "desc", "updated description"),
					resource.TestCheckResourceAttr("apisix_upstream.full", "nodes.#", "1"),
					resource.TestCheckResourceAttr("apisix_upstream.full", "nodes.0.weight", "3"),
				),
			},
			{
				Config:             testAccUpstreamFullUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "apisix_upstream.full",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
