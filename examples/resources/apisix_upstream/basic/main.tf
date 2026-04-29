# Basic Upstream Example

This example creates a simple upstream with a single node.

```hcl
terraform {
  required_providers {
    apisix = {
      source = "scicore/apisix"
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

resource "apisix_upstream" "basic" {
  name = "basic-upstream"
  type = "roundrobin"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}

output "upstream_id" {
  value = apisix_upstream.basic.id
}

output "upstream_name" {
  value = apisix_upstream.basic.name
}
