terraform {
  required_providers {
    apisix = {
      source  = "scicore/apisix"
      version = "0.1.0"
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

# Basic route with upstream_id
resource "apisix_upstream" "route_test" {
  name = "route-test-upstream"
  type = "roundrobin"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }

  labels = {
    test = "route"
  }
}

resource "apisix_route" "basic" {
  name = "test-route-basic"
  uri  = "/api/*"

  upstream_id = apisix_upstream.route_test.id

  status = 1
}

# Route with multiple URIs and hosts
resource "apisix_route" "advanced" {
  name  = "test-route-advanced"
  uris  = ["/api/*", "/v1/*"]
  hosts = ["api.example.com", "api.test.com"]
  methods = ["GET", "POST"]

  upstream_id = apisix_upstream.route_test.id

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }

  status = 1
}

# Route with vars filtering
resource "apisix_route" "with_vars" {
  name = "test-route-with-vars"
  uri  = "/admin/*"

  vars = jsonencode([
    ["http_method", "==", "GET"],
    ["remote_addr", "in", ["127.0.0.1", "10.0.0.1"]]
  ])

  upstream_id = apisix_upstream.route_test.id

  priority = 10
  status   = 1
}

# Route with all fields (comprehensive test)
resource "apisix_route" "complete" {
  name        = "test-route-complete"
  desc        = "Complete route with all supported fields"
  uris        = ["/complete/*"]
  hosts       = ["complete.example.com"]
  remote_addrs = ["10.0.0.0/8"]
  methods     = ["GET", "POST", "PUT"]
  priority    = 100
  status      = 1

  # Using inline upstream instead of upstream_id
  upstream {
    type = "roundrobin"

    nodes {
      host   = "127.0.0.1"
      port   = 8080
      weight = 100
    }
  }

  # Plugin configuration
  plugins = {
    "limit-count" = jsonencode({
      count         = 500
      time_window   = 60
      rejected_code = 429
    })
  }

  # Timeout configuration
  timeout {
    connect = 5
    send    = 10
    read    = 15
  }

  enable_websocket = true

  labels = {
    env        = "test"
    complexity = "complete"
    managed-by = "terraform"
  }
}

# Route with custom Lua script (alternative to plugins)
resource "apisix_route" "with_script" {
  name = "test-route-with-script"
  desc = "Route with custom Lua script"
  uri  = "/script/*"

  # Script must be a valid Lua module string
  # Note: Conflicts with `plugins` field - use one or the other
  script = <<-EOT
local _M = {}
function _M.access(conf, ctx)
    ngx.header["X-Custom-Route"] = "ScriptRoute"
    ngx.header["X-Request-ID"] = ngx.request_id()
end
return _M
EOT

  upstream_id = apisix_upstream.route_test.id
  status      = 1
}
