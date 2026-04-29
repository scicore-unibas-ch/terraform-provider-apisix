# Advanced Route Example

This example demonstrates a complete route configuration with all supported fields.

```hcl
terraform {
  required_providers {
    apisix = {
      source = "scicore-unibas-ch/apisix"
    }
  }
}

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
  timeout   = 30
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

resource "apisix_upstream" "backend" {
  name = "backend-upstream"
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

# Route with multiple match conditions
resource "apisix_route" "multi_match" {
  name  = "advanced-route"
  desc  = "Advanced route with multiple matching conditions"
  uris  = ["/api/*", "/v1/*", "/v2/*"]
  hosts = ["api.example.com", "api.test.com"]
  methods = ["GET", "POST", "PUT", "DELETE"]
  priority = 100

  upstream_id = apisix_upstream.backend.id

  status = 1
}

# Route with plugin configuration
resource "apisix_route" "with_plugins" {
  name = "route-with-plugins"
  uri  = "/protected/*"

  upstream_id = apisix_upstream.backend.id

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
    "cors" = jsonencode({
      allow_origins  = "*"
      allow_methods  = "*"
      allow_headers  = "*"
      expose_headers = ["Content-Type"]
      max_age        = 3600
    })
    "proxy-rewrite" = jsonencode({
      regex_uri = ["^/api/(.*)", "/v2/$1"]
    })
  }
}

# Route with vars filtering
resource "apisix_route" "with_vars" {
  name = "admin-route"
  uri  = "/admin/*"

  # Advanced filtering with vars
  vars = jsonencode([
    ["http_method", "==", "GET"],
    ["remote_addr", "in", ["127.0.0.1", "10.0.0.1", "192.168.0.0/16"]],
    ["http_host", "==", "admin.example.com"]
  ])

  upstream_id = apisix_upstream.backend.id
  priority    = 1000
}

# Route with inline upstream
resource "apisix_route" "inline_upstream" {
  name = "route-with-inline-upstream"
  uri  = "/service/*"

  upstream {
    type = "roundrobin"
    
    nodes {
      host   = "127.0.0.1"
      port   = 9000
      weight = 100
    }
    
    nodes {
      host   = "127.0.0.1"
      port   = 9001
      weight = 50
    }
  }

  timeout {
    connect = 5
    send    = 10
    read    = 30
  }

  enable_websocket = true
}

# Complete route with all fields
resource "apisix_route" "complete" {
  name          = "complete-route"
  desc          = "Complete route configuration with all fields"
  uris          = ["/api/v3/*"]
  hosts         = ["api.example.com"]
  methods       = ["GET", "POST"]
  priority      = 500
  status        = 1

  # Vars filtering
  vars = jsonencode([
    ["http_content_type", "==", "application/json"],
    ["http_authorization", "!=", ""]
  ])

  # Plugin configuration
  plugins = {
    "limit-count" = jsonencode({
      count         = 5000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
  }

  upstream_id = apisix_upstream.backend.id

  timeout {
    connect = 10
    send    = 30
    read    = 60
  }

  enable_websocket = true

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
    version    = "v3"
  }
}

# Route with custom Lua script (alternative to plugins)
resource "apisix_route" "with_script" {
  name = "route-with-script"
  desc = "Route with custom Lua script instead of plugins"
  uri  = "/custom/*"

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
  status      = 1

  labels = {
    env        = "production"
    auth-type  = "custom"
    managed-by = "terraform"
  }
}

# Outputs
output "route_ids" {
  value = {
    multi_match    = apisix_route.multi_match.id
    with_plugins   = apisix_route.with_plugins.id
    with_vars      = apisix_route.with_vars.id
    inline_upstream = apisix_route.inline_upstream.id
    complete       = apisix_route.complete.id
  }
}
