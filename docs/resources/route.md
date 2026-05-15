---
page_title: "apisix_route Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages an APISIX Route — the matching rule that maps incoming requests to an upstream or service.
---

# apisix_route

Manages an APISIX [Route](https://apisix.apache.org/docs/apisix/terminology/route/). A route binds matching predicates (URI, host, method, etc.) to a backend (an `upstream_id`, `service_id`, or inline `upstream` block) and an optional set of plugins.

## Example Usage

### Basic route

```hcl
resource "apisix_upstream" "backend" {
  id = "users-backend"
  nodes = [
    { host = "users.internal", port = 8080, weight = 100 },
  ]
}

resource "apisix_route" "users" {
  id  = "users-api"
  uri = "/users/*"

  upstream_id = apisix_upstream.backend.id
}
```

### Multi-URI route with plugins and host matching

```hcl
resource "apisix_route" "v2" {
  id      = "users-api-v2"
  uris    = ["/v2/users/*", "/v2/accounts/*"]
  hosts   = ["api.example.com", "api.test.example.com"]
  methods = ["GET", "POST", "PUT"]

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

### Advanced matching with `vars`

```hcl
resource "apisix_route" "admin" {
  id  = "admin-api"
  uri = "/admin/*"

  vars = jsonencode([
    ["http_method", "==", "GET"],
    ["remote_addr", "in", ["10.0.0.0/8"]]
  ])

  upstream_id = apisix_upstream.backend.id
  priority    = 100
}
```

### Inline upstream with timeouts

The inline `upstream` block accepts the full APISIX upstream surface (scheme, timeouts, retries, hashing, keepalive, mTLS, service discovery, health checks). See [`apisix_upstream`](upstream.md) for the complete attribute list.

```hcl
resource "apisix_route" "billing" {
  id      = "billing-api"
  uri     = "/billing/*"
  methods = ["GET", "POST"]

  upstream = {
    type   = "roundrobin"
    scheme = "https"
    nodes = [
      { host = "billing.internal", port = 443, weight = 100 },
    ]
  }

  timeout = {
    connect = 5
    send    = 10
    read    = 30
  }
}
```

## Argument Reference

- `id` — (Required, ForceNew) Unique route identifier. Changing this forces replacement.
- `name` — (Optional) Human-readable name.
- `desc` — (Optional) Description.
- `uri` / `uris` — (Optional, mutually exclusive) Path or paths to match.
- `host` / `hosts` — (Optional, mutually exclusive) Hostname or hostnames to match.
- `remote_addr` / `remote_addrs` — (Optional, mutually exclusive) Client IP or CIDR(s) to match.
- `methods` — (Optional) Set of HTTP methods. Valid values: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, `TRACE`, `CONNECT`, `PURGE`.
- `priority` — (Optional, Computed, Default `0`) Higher value matches first. Always populated in state after apply.
- `vars` — (Optional) JSON-encoded list of variable conditions for advanced routing. JSON-equivalent values are suppressed.
- `filter_func` — (Optional) Lua function source for custom filtering.
- `plugins` — (Optional) Map of plugin name to JSON-encoded configuration. Mutually exclusive with `script`. JSON-equivalent values are suppressed.
- `script` — (Optional) Lua script for custom logic. Mutually exclusive with `plugins`.
- `upstream_id` — (Optional) Reference to an `apisix_upstream` by id. Mutually exclusive with `upstream`.
- `upstream` — (Optional) Inline upstream definition. Mutually exclusive with `upstream_id`. Accepts the same attributes as the `apisix_upstream` resource (minus `id`).
- `service_id` — (Optional) Reference to an `apisix_service` by id.
- `plugin_config_id` — (Optional) Reference to an `apisix_plugin_config` by id.
- `labels` — (Optional) Map of string key/value pairs.
- `timeout` — (Optional) Per-route upstream timeout overrides (seconds). Object with optional `connect`, `send`, `read` integer fields.
- `enable_websocket` — (Optional, Computed, Default `false`) Enable WebSocket upgrade. Always populated in state after apply.
- `status` — (Optional, Computed, Default `1`) `1` = enabled, `0` = disabled. Always populated in state after apply.

## Import

```bash
terraform import apisix_route.users users-api
```
