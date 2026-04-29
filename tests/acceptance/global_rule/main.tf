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

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
  timeout   = 30
}

# Basic global rule
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

# Global rule with multiple plugins
resource "apisix_global_rule" "multi_plugins" {
  rule_id = "test-gr-multi"

  plugins = {
    "limit-count" = jsonencode({
      count         = 500
      time_window   = 60
      rejected_code = 503
    })
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "*"
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

# Global rule for route integration test
resource "apisix_global_rule" "route_integration" {
  rule_id = "test-gr-route"

  plugins = {
    "limit-count" = jsonencode({
      count         = 100
      time_window   = 60
      rejected_code = 429
    })
  }
}

# Upstream for route integration
resource "apisix_upstream" "test" {
  name = "test-gr-upstream"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}

# Route to verify global rules apply
resource "apisix_route" "with_global_rule" {
  name        = "test-route-with-gr"
  uri         = "/gr-test/*"
  upstream_id = apisix_upstream.test.id
  status      = 1
}
