# apisix_upstream

Manages an APISIX Upstream resource.

## Example Usage

### Basic Upstream

```hcl
resource "apisix_upstream" "basic" {
  name = "basic-upstream"
  type = "roundrobin"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}
```

### Upstream with Multiple Nodes

```hcl
resource "apisix_upstream" "multi_node" {
  name = "multi-node-upstream"
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

  nodes {
    host   = "10.0.1.12"
    port   = 8080
    weight = 50
  }
}
```

### Upstream with Health Checks

```hcl
resource "apisix_upstream" "with_healthcheck" {
  name = "upstream-with-healthcheck"
  type = "roundrobin"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }

  health_check = jsonencode({
    active = {
      http_path = "/health"
      interval  = 5
      timeout   = 3
      healthy = {
        interval  = 3
        successes = 2
      }
      unhealthy = {
        interval      = 3
        http_failures = 3
      }
    }
  })
}
```

### Upstream with Timeout Configuration

```hcl
resource "apisix_upstream" "with_timeout" {
  name = "upstream-with-timeout"
  type = "roundrobin"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }

  timeout {
    connect = 5
    send    = 10
    read    = 15
  }

  retries       = 3
  retry_timeout = 10
}
```

### Upstream with Service Discovery

```hcl
resource "apisix_upstream" "service_discovery" {
  name          = "upstream-with-discovery"
  type          = "roundrobin"
  service_name  = "my-service"
  discovery_type = "consul"

  discovery_args = {
    namespace_id = "production"
    group_name   = "backend"
  }
}
```

### Upstream with chash Load Balancing

```hcl
resource "apisix_upstream" "chash" {
  name    = "chash-upstream"
  type    = "chash"
  hash_on = "vars"
  key     = "remote_addr"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}
```

### Upstream with Keepalive Pool

```hcl
resource "apisix_upstream" "keepalive" {
  name = "upstream-with-keepalive"
  type = "roundrobin"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }

  keepalive_pool {
    size         = 320
    idle_timeout = 60
    requests     = 1000
  }
}
```

### Upstream with mTLS

```hcl
resource "apisix_upstream" "mtls" {
  name   = "upstream-with-mtls"
  type   = "roundrobin"
  scheme = "https"

  nodes {
    host   = "secure.example.com"
    port   = 443
    weight = 100
  }

  tls {
    client_cert = file("${path.module}/client.crt")
    client_key  = file("${path.module}/client.key")
    verify      = true
  }
}
```

### Complete Upstream with All Fields

```hcl
resource "apisix_upstream" "complete" {
  name          = "complete-upstream"
  desc          = "Complete upstream configuration with all fields"
  type          = "chash"
  scheme        = "http"
  hash_on       = "vars"
  key           = "remote_addr"
  pass_host     = "pass"
  retries       = 2
  retry_timeout = 5

  nodes {
    host     = "10.0.1.10"
    port     = 8080
    weight   = 100
    priority = 0
    metadata = {
      version = "v1"
      zone    = "us-east-1"
    }
  }

  nodes {
    host     = "10.0.1.11"
    port     = 8080
    weight   = 50
    priority = 1
  }

  timeout {
    connect = 3
    send    = 5
    read    = 10
  }

  health_check = jsonencode({
    active = {
      http_path     = "/health"
      interval      = 5
      timeout       = 3
      concurrency   = 10
      type          = "http"
      healthy = {
        interval      = 3
        successes     = 2
        http_statuses = [200, 302]
      }
      unhealthy = {
        interval      = 3
        http_failures = 3
        tcp_failures  = 2
        timeouts      = 3
        http_statuses = [429, 500, 502, 503, 504]
      }
    }
    passive = {
      type = "http"
      healthy = {
        http_statuses = [200, 201, 202, 301, 302]
        successes     = 5
      }
      unhealthy = {
        http_failures = 5
        tcp_failures  = 2
        timeouts      = 7
        http_statuses = [429, 500, 503]
      }
    }
  })

  keepalive_pool {
    size         = 320
    idle_timeout = 60
    requests     = 1000
  }

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Optional) Name of the upstream.
- `desc` - (Optional) Description of the upstream.
- `type` - (Optional) Load balancing algorithm. Valid values: `roundrobin`, `chash`, `ewma`, `least_conn`. Defaults to `roundrobin`.
- `scheme` - (Optional) Scheme to use when communicating with the upstream. Valid values: `grpc`, `grpcs`, `http`, `https`, `tcp`, `tls`, `udp`, `kafka`. Defaults to `http`.

### Nodes

- `nodes` - (Required) List of upstream nodes. Each node supports:
  - `host` - (Required) Hostname or IP of the node.
  - `port` - (Optional) Port of the node.
  - `weight` - (Optional) Weight of the node for load balancing. Defaults to `1`.
  - `priority` - (Optional) Priority of the node. Nodes with lower priority are tried first. Defaults to `0`.
  - `metadata` - (Optional) Metadata for the node as key-value pairs.

### Timeout

- `timeout` - (Optional) Timeout configuration block:
  - `connect` - (Optional) Connect timeout in seconds.
  - `send` - (Optional) Send timeout in seconds.
  - `read` - (Optional) Read timeout in seconds.

### Health Check

- `health_check` - (Optional) JSON-encoded health check configuration. Supports both active and passive health checks.

### Retry Configuration

- `retries` - (Optional) Number of retries for the upstream.
- `retry_timeout` - (Optional) Timeout for retries in seconds.

### Load Balancing

- `hash_on` - (Optional) Hash on parameter for chash load balancing. Valid values: `vars`, `header`, `cookie`, `consumer`, `vars_combinations`. Defaults to `vars`.
- `key` - (Optional) The key for chash load balancing (e.g., `remote_addr`, `uri`, `arg_name`).

### Host Configuration

- `pass_host` - (Optional) Mode of host passing. Valid values: `pass`, `node`, `rewrite`. Defaults to `pass`.
- `upstream_host` - (Optional) Custom host for the upstream request. Required when `pass_host` is `rewrite`.

### Service Discovery

- `service_name` - (Optional) Service name for service discovery. Required when using service discovery.
- `discovery_type` - (Optional) Type of service discovery.
- `discovery_args` - (Optional) Arguments for service discovery:
  - `namespace_id` - (Optional) Namespace ID.
  - `group_name` - (Optional) Group name.

### Keepalive Pool

- `keepalive_pool` - (Optional) Keepalive pool configuration block:
  - `size` - (Optional) Size of the keepalive pool. Defaults to `320`.
  - `idle_timeout` - (Optional) Idle timeout for keepalive connections in seconds. Defaults to `60`.
  - `requests` - (Optional) Maximum number of requests per connection. Defaults to `1000`.

### TLS/mTLS

- `tls` - (Optional) TLS client certificate configuration block:
  - `client_cert` - (Optional, Sensitive) Client certificate content for mTLS.
  - `client_key` - (Optional, Sensitive) Client private key content for mTLS.
  - `client_cert_id` - (Optional) Reference to SSL object for client certificate.
  - `verify` - (Optional) Enable server certificate verification. Defaults to `false`.

### Labels

- `labels` - (Optional) Labels for the upstream as key-value pairs.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The ID of the upstream.

## Import

APISIX Upstreams can be imported using the upstream ID, e.g.,

```bash
tofu import apisix_upstream.example <upstream-id>
```

Example:

```bash
tofu import apisix_upstream.example test-upstream-1
```
