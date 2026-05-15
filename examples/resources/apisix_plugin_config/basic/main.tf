terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "~> 0.2"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

# Reusable plugin bundle that routes can attach via plugin_config_id
resource "apisix_plugin_config" "basic" {
  id   = "basic-rate-limit"
  desc = "Basic rate limiting configuration"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}
