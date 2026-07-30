terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "~> 0.5"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

resource "apisix_upstream" "backend" {
  id   = "advanced-service-backend"
  type = "roundrobin"

  nodes = [
    { host = "10.0.1.10", port = 8080, weight = 100 },
    { host = "10.0.1.11", port = 8080, weight = 50 },
  ]
}

# Service with inline upstream (attribute syntax), hosts, plugins, labels
resource "apisix_service" "advanced" {
  id    = "advanced-service"
  name  = "advanced-service"
  desc  = "Advanced service with all features"
  hosts = ["api.example.com", "api.test.com"]

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
    })
  }

  upstream = {
    type = "roundrobin"
    nodes = [
      { host = "10.0.1.10", port = 8080, weight = 100 },
      { host = "10.0.1.11", port = 8080, weight = 50 },
    ]
  }

  enable_websocket = true

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}

# Service with custom Lua script (mutually exclusive with `plugins`)
resource "apisix_service" "with_script" {
  id   = "advanced-service-with-script"
  name = "service-with-script"
  desc = "Service with custom Lua script instead of plugins"

  script = <<-EOT
    local _M = {}
    function _M.access(conf, ctx)
        ngx.header["X-Custom-Header"] = "CustomValue"
        ngx.header["X-Request-ID"] = ngx.request_id()
    end
    return _M
  EOT

  upstream_id = apisix_upstream.backend.id

  labels = {
    env        = "production"
    auth-type  = "custom"
    managed-by = "terraform"
  }
}
