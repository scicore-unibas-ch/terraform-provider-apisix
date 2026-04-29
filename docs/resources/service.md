# apisix_service

Manages an APISIX Service resource.

## Example Usage

### Basic Service

```hcl
resource "apisix_service" "basic" {
  name = "basic-service"
  desc = "Basic service for testing"

  upstream_id = apisix_upstream.backend.id
}
```

### Service with Host Matching

```hcl
resource "apisix_service" "with_hosts" {
  name = "service-with-hosts"
  desc = "Service with host matching"

  hosts = ["api.example.com", "api.test.com"]

  upstream_id = apisix_upstream.backend.id
}
```

### Service with Plugin Configuration

```hcl
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

  upstream_id = apisix_upstream.backend.id
}
```

### Service with Inline Upstream

```hcl
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
```

### Service with Labels and Websocket

```hcl
resource "apisix_service" "with_labels" {
  name = "service-with-labels"
  desc = "Service with labels and websocket support"

  upstream_id = apisix_upstream.backend.id

  enable_websocket = true

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}
```

### Complete Service with All Fields

```hcl
resource "apisix_service" "complete" {
  name = "complete-service"
  desc = "Complete service configuration with all fields"

  hosts = ["api.example.com"]

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
```

## Argument Reference

The following arguments are supported:

- `name` - (Optional) Name of the service.
- `desc` - (Optional) Description of the service.
- `hosts` - (Optional) List of hosts to match.
- `plugins` - (Optional) Plugin configurations as a map of JSON-encoded strings. Conflicts with `script`.
- `script` - (Optional) Lua script for custom logic. Conflicts with `plugins`.
- `upstream_id` - (Optional) ID of the upstream resource. Conflicts with `upstream`.
- `upstream` - (Optional) Inline upstream configuration block. Conflicts with `upstream_id`.
  - `type` - (Optional) Load balancing type. Defaults to `roundrobin`.
  - `nodes` - (Required) List of upstream nodes.
    - `host` - (Required) Node host.
    - `port` - (Required) Node port.
    - `weight` - (Optional) Node weight. Defaults to `1`.
- `labels` - (Optional) Labels as key-value pairs.
- `enable_websocket` - (Optional) Enable websocket support. Defaults to `false`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The ID of the service.

## Import

APISIX Services can be imported using the service ID, e.g.,

```bash
tofu import apisix_service.example <service-id>
```

Example:

```bash
tofu import apisix_service.example test-service-1
```
