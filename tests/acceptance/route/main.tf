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

resource "apisix_upstream" "route_test" {
  id   = "route-test-upstream"
  name = "route-test-upstream"
  type = "roundrobin"

  nodes = [
    {
      host   = "127.0.0.1"
      port   = 8080
      weight = 100
    },
  ]

  labels = {
    test = "route"
  }
}

resource "apisix_route" "basic" {
  id  = "test-route-basic"
  uri = "/api/*"

  upstream_id = apisix_upstream.route_test.id

  status = 1
}

resource "apisix_route" "advanced" {
  id      = "test-route-advanced"
  uris    = ["/api/*", "/v1/*"]
  hosts   = ["api.example.com", "api.test.com"]
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

resource "apisix_route" "with_vars" {
  id  = "test-route-with-vars"
  uri = "/admin/*"

  vars = jsonencode([
    ["http_method", "==", "GET"],
    ["remote_addr", "in", ["127.0.0.1", "10.0.0.1"]]
  ])

  upstream_id = apisix_upstream.route_test.id

  priority = 10
  status   = 1
}

# Comprehensive route exercising the full inline upstream surface and timeouts.
resource "apisix_route" "complete" {
  id           = "test-route-complete"
  desc         = "Complete route with all supported fields"
  uris         = ["/complete/*"]
  hosts        = ["complete.example.com"]
  remote_addrs = ["10.0.0.0/8"]
  methods      = ["GET", "POST", "PUT"]
  priority     = 100
  status       = 1

  upstream = {
    type = "roundrobin"
    nodes = [
      {
        host   = "127.0.0.1"
        port   = 8080
        weight = 100
      },
    ]
  }

  plugins = {
    "limit-count" = jsonencode({
      count         = 500
      time_window   = 60
      rejected_code = 429
    })
  }

  timeout = {
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

# Route exercising the script field (mutually exclusive with plugins).
resource "apisix_route" "with_script" {
  id   = "test-route-with-script"
  desc = "Route with custom Lua script"
  uri  = "/script/*"

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
