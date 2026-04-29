# apisix_route

Manages an APISIX Route resource.

## Example Usage

### Basic Route

```hcl
resource "apisix_route" "basic" {
  name = "basic-route"
  uri  = "/api/*"

  upstream_id = apisix_upstream.backend.id

  status = 1
}
```

### Route with Multiple URIs and Hosts

```hcl
resource "apisix_route" "multi" {
  name  = "multi-match-route"
  uris  = ["/api/*", "/v1/*"]
  hosts = ["api.example.com", "api.test.com"]
  methods = ["GET", "POST"]

  upstream_id = apisix_upstream.backend.id
}
```

### Route with Plugin Configuration

```hcl
resource "apisix_route" "with_plugins" {
  name = "route-with-plugins"
  uri  = "/api/*"

  upstream_id = apisix_upstream.backend.id

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
}
```

### Route with Vars Filtering

```hcl
resource "apisix_route" "with_vars" {
  name = "advanced-route"
  uri  = "/admin/*"

  # Advanced filtering with vars
  vars = jsonencode([
    ["http_method", "==", "GET"],
    ["remote_addr", "in", ["127.0.0.1", "10.0.0.1"]],
    ["http_host", "==", "admin.example.com"]
  ])

  upstream_id = apisix_upstream.backend.id
  priority    = 10
}
```

### Route with Inline Upstream

```hcl
resource "apisix_route" "inline_upstream" {
  name = "route-with-inline-upstream"
  uri  = "/service/*"

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
```

### Route with Timeout Configuration

```hcl
resource "apisix_route" "with_timeout" {
  name = "route-with-timeout"
  uri  = "/slow-api/*"

  upstream_id = apisix_upstream.backend.id

  timeout {
    connect = 10
    send    = 30
    read    = 60
  }
}
```

### Complete Route with All Fields

```hcl
resource "apisix_route" "complete" {
  name          = "complete-route"
  desc          = "Complete route configuration with all fields"
  uris          = ["/api/*", "/v2/*"]
  hosts         = ["api.example.com"]
  methods       = ["GET", "POST", "PUT", "DELETE"]
  priority      = 100
  status        = 1

  # Advanced vars filtering
  vars = jsonencode([
    ["http_method", "==", "POST"],
    ["http_content_type", "==", "application/json"],
    ["remote_addr", "in", ["10.0.0.0/8"]]
  ])

  # Plugin configuration
  plugins = {
    "limit-count" = jsonencode({
      count         = 5000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
    "proxy-rewrite" = jsonencode({
      regex_uri = ["^/api/(.*)", "/v2/$1"]
    })
  }

  upstream_id = apisix_upstream.backend.id

  timeout {
    connect = 5
    send    = 10
    read    = 30
  }

  enable_websocket = true

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Optional) Name of the route.
- `desc` - (Optional) Description of the route.
- `uri` - (Optional) Request URI prefix. Conflicts with `uris`.
- `uris` - (Optional) Request URI prefixes as a list. Conflicts with `uri`.
- `host` - (Optional) Request host. Conflicts with `hosts`.
- `hosts` - (Optional) Request hosts as a list. Conflicts with `host`.
- `remote_addr` - (Optional) Client IP. Conflicts with `remote_addrs`.
- `remote_addrs` - (Optional) Client IPs as a list. Conflicts with `remote_addr`.
- `methods` - (Optional) HTTP methods. Valid values: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, `TRACE`, `CONNECT`, `PURGE`.
- `priority` - (Optional) Priority of the route. Higher value means higher priority. Defaults to `0`.
- `vars` - (Optional) JSON-encoded filter expressions for advanced routing rules.
- `filter_func` - (Optional) Lua function for custom filtering logic.
- `plugins` - (Optional) Plugin configurations as a map of JSON-encoded strings.
- `script` - (Optional) Lua script for custom logic. Conflicts with `plugins`.
- `upstream_id` - (Optional) ID of the upstream resource. Conflicts with `upstream`.
- `upstream` - (Optional) Inline upstream configuration block. Conflicts with `upstream_id`.
  - `type` - (Optional) Load balancing type. Defaults to `roundrobin`.
  - `nodes` - (Required) List of upstream nodes.
    - `host` - (Required) Node host.
    - `port` - (Required) Node port.
    - `weight` - (Optional) Node weight. Defaults to `1`.
- `service_id` - (Optional) ID of the service resource.
- `plugin_config_id` - (Optional) ID of the plugin configuration.
- `labels` - (Optional) Labels as key-value pairs.
- `timeout` - (Optional) Timeout configuration block:
  - `connect` - (Optional) Connect timeout in seconds.
  - `send` - (Optional) Send timeout in seconds.
  - `read` - (Optional) Read timeout in seconds.
- `enable_websocket` - (Optional) Enable websocket support. Defaults to `false`.
- `status` - (Optional) Route status. `1`=enabled, `0`=disabled. Defaults to `1`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The ID of the route.

## Import

APISIX Routes can be imported using the route ID, e.g.,

```bash
tofu import apisix_route.example <route-id>
```

Example:

```bash
tofu import apisix_route.example test-route-1
```
