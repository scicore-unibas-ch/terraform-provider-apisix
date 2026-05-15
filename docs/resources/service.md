---
page_title: "apisix_service Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages an APISIX Service — a reusable bundle of host/plugin/upstream configuration that routes can reference.
---

# apisix_service

Manages an APISIX [Service](https://apisix.apache.org/docs/apisix/terminology/service/). Services let multiple routes share the same upstream, hosts, and plugin configuration. A route attaches to a service via `service_id`.

## Example Usage

### Service referencing an upstream by id

```hcl
resource "apisix_upstream" "users" {
  id = "users-upstream"
  nodes = [
    { host = "users-1.internal", port = 8080, weight = 100 },
    { host = "users-2.internal", port = 8080, weight = 100 },
  ]
}

resource "apisix_service" "users_api" {
  id    = "users-api"
  name  = "Users API"
  hosts = ["api.example.com"]

  upstream_id = apisix_upstream.users.id

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
  }

  enable_websocket = false

  labels = {
    env  = "production"
    team = "platform"
  }
}
```

### Service with an inline upstream

The inline `upstream` block accepts the full APISIX upstream surface — see [`apisix_upstream`](upstream.md) for the complete attribute list.

```hcl
resource "apisix_service" "billing_api" {
  id   = "billing-api"
  name = "Billing API"

  upstream = {
    type   = "roundrobin"
    scheme = "https"
    nodes = [
      { host = "billing.internal", port = 443, weight = 100 },
    ]
    timeout = {
      connect = 5
      send    = 10
      read    = 30
    }
  }
}
```

## Argument Reference

- `id` — (Required, ForceNew) Unique identifier for the service. Changing this forces replacement.
- `name` — (Optional) Human-readable name.
- `desc` — (Optional) Description.
- `hosts` — (Optional) Set of hostnames to match.
- `plugins` — (Optional) Map of plugin name to JSON-encoded configuration. Mutually exclusive with `script`. JSON-equivalent values are suppressed by the provider.
- `script` — (Optional) Lua script for custom logic. Mutually exclusive with `plugins`.
- `upstream_id` — (Optional) Reference to an `apisix_upstream` by id. Mutually exclusive with `upstream`.
- `upstream` — (Optional) Inline upstream definition. Mutually exclusive with `upstream_id`. Accepts the same attributes as the `apisix_upstream` resource (minus `id`).
- `labels` — (Optional) Map of string key/value pairs.
- `enable_websocket` — (Optional, Computed, Default `false`) Enable WebSocket upgrade. Always populated in state after apply.

## Import

```bash
terraform import apisix_service.users_api users-api
```
