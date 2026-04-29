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

# Global rule with multiple plugins
resource "apisix_global_rule" "multi_plugin" {
  rule_id = "global-security"

  plugins = {
    "limit-count" = jsonencode({
      count         = 5000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "*"
      allow_headers = "*"
    })
  }
}

# Global IP restriction
resource "apisix_global_rule" "ip_restriction" {
  rule_id = "global-ip-restriction"

  plugins = {
    "ip-restriction" = jsonencode({
      blacklist = ["127.0.0.1"]
    })
  }
}
