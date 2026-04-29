terraform {
  required_providers {
    apisix = {
      source  = "scicore/apisix"
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

# Basic service
resource "apisix_service" "basic" {
  name = "basic-service"
  desc = "Basic service for testing"

  upstream_id = apisix_upstream.test.id
}

# Service with hosts
resource "apisix_service" "with_hosts" {
  name = "service-with-hosts"
  desc = "Service with host matching"

  hosts = ["api.example.com", "api.test.com"]

  upstream_id = apisix_upstream.test.id
}

# Service with plugins
resource "apisix_service" "with_plugins" {
  name = "service-with-plugins"
  desc = "Service with plugin configuration"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "*"
    })
  }

  upstream_id = apisix_upstream.test.id
}

# Service with inline upstream
resource "apisix_service" "with_upstream" {
  name = "service-with-upstream"
  desc = "Service with inline upstream"

  upstream {
    type = "roundrobin"

    nodes {
      host   = "127.0.0.1"
      port   = 8080
      weight = 100
    }

    nodes {
      host   = "127.0.0.1"
      port   = 8081
      weight = 50
    }
  }
}

# Service with labels
resource "apisix_service" "with_labels" {
  name = "service-with-labels"
  desc = "Service with labels"

  upstream_id = apisix_upstream.test.id

  enable_websocket = true

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}

# Service with script (custom Lua logic)
resource "apisix_service" "with_script" {
  name = "service-with-script"
  desc = "Service with custom Lua script"

  # Script must be a valid Lua module string
  script = <<-EOT
local _M = {}
function _M.access(conf, ctx)
    ngx.header["X-Custom-Header"] = "CustomValue"
end
return _M
EOT

  upstream_id = apisix_upstream.test.id
}

# Shared upstream for services
resource "apisix_upstream" "test" {
  name = "service-test-upstream"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}
