terraform {
  required_providers {
    apisix = {
      source  = "scicore/apisix"
      version = "0.1.0"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

# Basic global rule
# Note: This resource is fully implemented but acceptance tests are not executed.
resource "apisix_global_rule" "basic" {
  rule_id = "global-rate-limit"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}
