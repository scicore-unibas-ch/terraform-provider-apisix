terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "0.1.0"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

# Plugin config with multiple plugins
resource "apisix_plugin_config" "api_protection" {
  config_id = "api-protection"
  desc      = "API protection with rate limiting and CORS"

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

  labels = {
    env        = "production"
    team       = "security"
    managed-by = "terraform"
  }
}

# Route using the plugin config
resource "apisix_upstream" "backend" {
  name = "backend-upstream"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}

resource "apisix_route" "protected_route" {
  name             = "protected-route"
  uri              = "/api/*"
  plugin_config_id = apisix_plugin_config.api_protection.config_id
  upstream_id      = apisix_upstream.backend.id
}
