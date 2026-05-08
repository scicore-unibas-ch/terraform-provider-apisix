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
  timeout   = 30
}

resource "apisix_upstream" "test" {
  id   = "service-test-upstream"
  name = "service-test-upstream"

  nodes = [
    {
      host   = "127.0.0.1"
      port   = 8080
      weight = 100
    },
  ]
}

resource "apisix_service" "basic" {
  id   = "test-service-basic"
  name = "basic-service"
  desc = "Basic service for testing"

  upstream_id = apisix_upstream.test.id
}

resource "apisix_service" "with_hosts" {
  id   = "test-service-with-hosts"
  name = "service-with-hosts"
  desc = "Service with host matching"

  hosts = ["api.example.com", "api.test.com"]

  upstream_id = apisix_upstream.test.id
}

resource "apisix_service" "with_plugins" {
  id   = "test-service-with-plugins"
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

resource "apisix_service" "with_upstream" {
  id   = "test-service-with-upstream"
  name = "service-with-upstream"
  desc = "Service with inline upstream"

  upstream = {
    type = "roundrobin"
    nodes = [
      {
        host   = "127.0.0.1"
        port   = 8080
        weight = 100
      },
      {
        host   = "127.0.0.1"
        port   = 8081
        weight = 50
      },
    ]
  }
}

resource "apisix_service" "with_labels" {
  id   = "test-service-with-labels"
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

resource "apisix_service" "with_script" {
  id   = "test-service-with-script"
  name = "service-with-script"
  desc = "Service with custom Lua script"

  script = <<-EOT
local _M = {}
function _M.access(conf, ctx)
    ngx.header["X-Custom-Header"] = "CustomValue"
end
return _M
EOT

  upstream_id = apisix_upstream.test.id
}
