# Advanced Upstream Example

This example demonstrates a complete upstream configuration with all supported fields.

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

# Complete upstream with all fields
resource "apisix_upstream" "advanced" {
  name          = "advanced-upstream"
  desc          = "Advanced upstream with all configuration options"
  type          = "chash"
  scheme        = "http"
  hash_on       = "vars"
  key           = "remote_addr"
  pass_host     = "pass"
  retries       = 2
  retry_timeout = 5

  # Multiple nodes with different weights and priorities
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

  # Timeout configuration
  timeout {
    connect = 3
    send    = 5
    read    = 10
  }

  # Active and passive health checks
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

  # Keepalive pool configuration
  keepalive_pool {
    size         = 320
    idle_timeout = 60
    requests     = 1000
  }

  # Labels for organization
  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}

# Upstream with service discovery
resource "apisix_upstream" "consul" {
  name           = "consul-upstream"
  type           = "roundrobin"
  service_name   = "api-service"
  discovery_type = "consul"

  discovery_args = {
    namespace_id = "production"
    group_name   = "backend"
  }
}

# Upstream with mTLS
resource "apisix_upstream" "mtls" {
  name   = "mtls-upstream"
  type   = "roundrobin"
  scheme = "https"

  nodes {
    host   = "secure.example.com"
    port   = 443
    weight = 100
  }

  tls {
    client_cert = file("${path.module}/certs/client.crt")
    client_key  = file("${path.module}/certs/client.key")
    verify      = true
  }
}

# Output the upstream IDs
output "advanced_upstream_id" {
  value = apisix_upstream.advanced.id
}

output "consul_upstream_id" {
  value = apisix_upstream.consul.id
}

output "mtls_upstream_id" {
  value = apisix_upstream.mtls.id
}
