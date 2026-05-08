terraform {
  required_providers {
    apisix = {
      source = "scicore-unibas-ch/apisix"
    }
  }
}

variable "apisix_base_url" {
  type    = string
  default = "http://localhost:9180/apisix/admin"
}

variable "apisix_admin_key" {
  type      = string
  default   = "test123456789"
  sensitive = true
}

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
}

resource "apisix_global_rule" "basic" {
  id = "test-gr-basic"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}

resource "apisix_global_rule" "multi_plugins" {
  id = "test-gr-multi"

  plugins = {
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "*"
    })
    "limit-req" = jsonencode({
      key   = "remote_addr"
      rate  = 100
      burst = 50
    })
  }
}

resource "apisix_global_rule" "ip_restriction" {
  id = "test-gr-ip"

  plugins = {
    "ip-restriction" = jsonencode({
      blacklist = ["127.0.0.1"]
    })
  }
}

resource "apisix_global_rule" "route_integration" {
  id = "test-gr-route"

  plugins = {
    "response-rewrite" = jsonencode({
      headers = {
        "X-Test-Header" = "global-rule-test"
      }
    })
  }
}

resource "apisix_upstream" "test" {
  id   = "test-gr-upstream"
  name = "test-gr-upstream"
  type = "roundrobin"

  nodes = [
    {
      host   = "127.0.0.1"
      port   = 8080
      weight = 100
    },
  ]
}

resource "apisix_route" "with_global_rule" {
  id          = "test-route-with-gr"
  name        = "test-route-with-gr"
  uri         = "/gr-test/*"
  upstream_id = apisix_upstream.test.id
  status      = 1
}
