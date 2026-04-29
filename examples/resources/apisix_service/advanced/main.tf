terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "0.1.0"
    }
  }
}

provider "apisix" {
  api_key = "test123456789"
}

resource "apisix_upstream" "backend" {
  name = "backend-upstream"

  nodes {
    host   = "10.0.1.10"
    port   = 8080
    weight = 100
  }

  nodes {
    host   = "10.0.1.11"
    port   = 8080
    weight = 50
  }
}

resource "apisix_service" "advanced" {
  name = "advanced-service"
  desc = "Advanced service with all features"

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

  upstream {
    type = "roundrobin"

    nodes {
      host   = "10.0.1.10"
      port   = 8080
      weight = 100
    }

    nodes {
      host   = "10.0.1.11"
      port   = 8080
      weight = 50
    }
  }

  enable_websocket = true

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}

# Service with custom Lua script (alternative to plugins)
resource "apisix_service" "with_script" {
  name = "service-with-script"
  desc = "Service with custom Lua script instead of plugins"

  # Script must be a valid Lua module string
  # Note: Conflicts with `plugins` field - use one or the other
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
