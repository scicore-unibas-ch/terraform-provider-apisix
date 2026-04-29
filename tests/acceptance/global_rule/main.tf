terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "0.1.0"
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

# Basic global rule with limit-count
resource "apisix_global_rule" "basic" {
  rule_id = "test-gr-basic"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}

# Global rule with multiple plugins (cors + limit-req instead of limit-count)
resource "apisix_global_rule" "multi_plugins" {
  rule_id = "test-gr-multi"

  plugins = {
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "*"
    })
    "limit-req" = jsonencode({
      rate = 100
      burst = 50
    })
  }
}

# Global rule with IP restriction
resource "apisix_global_rule" "ip_restriction" {
  rule_id = "test-gr-ip"

  plugins = {
    "ip-restriction" = jsonencode({
      blacklist = ["127.0.0.1"]
    })
  }
}

# Global rule for route integration test (uses response-rewrite instead of limit-count)
resource "apisix_global_rule" "route_integration" {
  rule_id = "test-gr-route"

  plugins = {
    "response-rewrite" = jsonencode({
      headers = {
        "X-Test-Header" = "global-rule-test"
      }
    })
  }
}

# Upstream for route integration test
resource "apisix_upstream" "test" {
  name = "test-gr-upstream"
  type = "roundrobin"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}

# Route for integration test
resource "apisix_route" "with_global_rule" {
  name        = "test-route-with-gr"
  uri         = "/gr-test/*"
  upstream_id = apisix_upstream.test.id
  status      = 1
}

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
}
