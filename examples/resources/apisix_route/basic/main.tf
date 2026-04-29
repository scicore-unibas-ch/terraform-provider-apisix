# Basic Route Example

This example creates a simple route with minimal configuration.

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
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}

resource "apisix_route" "basic" {
  name = "basic-route"
  uri  = "/api/*"

  upstream_id = apisix_upstream.backend.id

  status = 1
}

output "route_id" {
  value = apisix_route.basic.id
}

output "route_name" {
  value = apisix_route.basic.name
}
