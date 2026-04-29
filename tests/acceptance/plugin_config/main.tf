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

# Basic plugin config
resource "apisix_plugin_config" "basic" {
  config_id = "test-pc-basic"
  desc      = "Basic plugin config for testing"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}

# Plugin config with multiple plugins
resource "apisix_plugin_config" "multi_plugins" {
  config_id = "test-pc-multi"
  desc      = "Plugin config with multiple plugins"

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

# Plugin config with labels
resource "apisix_plugin_config" "with_labels" {
  config_id = "test-pc-labels"
  desc      = "Plugin config with labels"

  plugins = {
    "limit-count" = jsonencode({
      count         = 2000
      time_window   = 60
      rejected_code = 429
    })
  }

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}

# Plugin config for route integration test
resource "apisix_plugin_config" "route_integration" {
  config_id = "test-pc-route"
  desc      = "Plugin config for route integration"

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
  name = "test-pc-upstream"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}

# Route using plugin config
resource "apisix_route" "with_plugin_config" {
  name             = "test-route-with-pc"
  uri              = "/pc-test/*"
  plugin_config_id = apisix_plugin_config.route_integration.config_id
  upstream_id      = apisix_upstream.test.id
  status           = 1
}
