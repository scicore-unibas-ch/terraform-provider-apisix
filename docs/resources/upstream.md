---
page_title: "apisix_upstream Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages an APISIX Upstream — a backend definition (one or more nodes, or a service-discovery reference) used by routes and services.
---

# apisix_upstream

Manages an APISIX [Upstream](https://apisix.apache.org/docs/apisix/terminology/upstream/). An upstream is a named collection of backend nodes (or a service-discovery reference) along with load-balancing, timeout, keepalive, and mTLS settings. Routes and services reference an upstream by id.

## Example Usage

### Basic upstream

```hcl
resource "apisix_upstream" "users" {
  id   = "users-backend"
  name = "Users service"

  nodes = [
    { host = "users-1.internal", port = 8080, weight = 100 },
    { host = "users-2.internal", port = 8080, weight = 100 },
  ]
}
```

### Round-robin with timeouts and retries

```hcl
resource "apisix_upstream" "billing" {
  id            = "billing-backend"
  scheme        = "https"
  retries       = 3
  retry_timeout = 10

  nodes = [
    { host = "billing-1.internal", port = 443, weight = 80 },
    { host = "billing-2.internal", port = 443, weight = 20 },
  ]

  timeout = {
    connect = 5
    send    = 10
    read    = 30
  }
}
```

### Consistent hashing on client IP

```hcl
resource "apisix_upstream" "cache" {
  id      = "cache-backend"
  type    = "chash"
  hash_on = "vars"
  key     = "remote_addr"

  nodes = [
    { host = "cache-1.internal", port = 11211, weight = 1 },
    { host = "cache-2.internal", port = 11211, weight = 1 },
    { host = "cache-3.internal", port = 11211, weight = 1 },
  ]
}
```

### Active health checks

```hcl
resource "apisix_upstream" "checked" {
  id = "checked-backend"

  nodes = [
    { host = "app-1.internal", port = 8080, weight = 100 },
    { host = "app-2.internal", port = 8080, weight = 100 },
  ]

  health_check = jsonencode({
    active = {
      http_path = "/healthz"
      interval  = 5
      timeout   = 3
      type      = "http"
      healthy   = { interval = 3, successes = 2 }
      unhealthy = { interval = 3, http_failures = 3 }
    }
  })
}
```

### mTLS to the upstream

```hcl
resource "apisix_upstream" "mtls_backend" {
  id     = "mtls-backend"
  scheme = "https"

  nodes = [
    { host = "secure.internal", port = 8443, weight = 100 },
  ]

  tls = {
    client_cert = file("${path.module}/client.crt")
    client_key  = file("${path.module}/client.key")
  }
}
```

### Service discovery (e.g. Consul)

```hcl
resource "apisix_upstream" "discovered" {
  id             = "discovered-backend"
  service_name   = "users-service"
  discovery_type = "consul"

  discovery_args = {
    namespace = "production"
  }
}
```

## Argument Reference

### Top-level

- `id` — (Required, ForceNew) Unique upstream identifier. Changing this forces replacement.
- `name` — (Optional) Human-readable name.
- `desc` — (Optional) Description.
- `type` — (Optional, Default `roundrobin`) Load-balancing algorithm. One of `roundrobin`, `chash`, `ewma`, `least_conn`.
- `nodes` — (Optional) List of backend nodes. Required when not using service discovery. See [Nodes](#nodes) below.
- `health_check` — (Optional) JSON-encoded active/passive health-check configuration. JSON-equivalent values are suppressed.
- `timeout` — (Optional) Connect/send/read timeout overrides (seconds). Object with optional `connect`, `send`, `read` integer fields.
- `retries` — (Optional) Number of retry attempts.
- `retry_timeout` — (Optional) Total retry timeout in seconds.
- `scheme` — (Optional, Default `http`) Upstream protocol. One of `grpc`, `grpcs`, `http`, `https`, `tcp`, `tls`, `udp`, `kafka`.
- `labels` — (Optional) Map of string key/value pairs.
- `service_name` — (Optional) Service name for service discovery (alternative to `nodes`).
- `discovery_type` — (Optional) Service discovery type (e.g. `consul`, `nacos`, `eureka`).
- `discovery_args` — (Optional) Map of service-discovery arguments (e.g. `namespace_id`, `group_name`).
- `hash_on` — (Optional, Default `vars`) Source of the hashing key for `chash`. One of `vars`, `header`, `cookie`, `consumer`, `vars_combinations`.
- `key` — (Optional) Hashing key for `chash` (e.g. `remote_addr`, `uri`, `arg_name`).
- `pass_host` — (Optional, Default `pass`) How to set the upstream `Host` header. One of `pass`, `node`, `rewrite`.
- `upstream_host` — (Optional) Custom `Host` header. Required when `pass_host` is `rewrite`.
- `keepalive_pool` — (Optional) Keepalive pool configuration. See [Keepalive Pool](#keepalive-pool) below.
- `tls` — (Optional) TLS configuration for mTLS to the upstream. See [TLS](#tls) below.

### Nodes

Each entry in `nodes` is an object with:

- `host` — (Required) Node hostname or IP.
- `port` — (Required) Node port (1-65535).
- `weight` — (Optional, Default `1`) Load-balancing weight.
- `priority` — (Optional, Default `0`) Lower priority is tried first.
- `metadata` — (Optional) Map of per-node metadata key/value pairs.

### Keepalive Pool

- `size` — (Optional, Default `320`) Pool size.
- `idle_timeout` — (Optional, Default `60`) Idle timeout in seconds.
- `requests` — (Optional, Default `1000`) Max requests per connection.

### TLS

- `client_cert` — (Optional, Sensitive) Client certificate (PEM).
- `client_key` — (Optional, Sensitive) Client private key (PEM).
- `client_cert_id` — (Optional) Reference to an SSL object (alternative to inline cert/key).
- `verify` — (Optional, Default `false`) Verify the server certificate. Currently only effective for `kafka` upstreams.

## Import

```bash
terraform import apisix_upstream.users users-backend
```
