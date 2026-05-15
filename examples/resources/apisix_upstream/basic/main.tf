terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "~> 0.2"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

resource "apisix_upstream" "basic" {
  id   = "basic-upstream"
  name = "basic-upstream"
  type = "roundrobin"

  nodes = [
    { host = "127.0.0.1", port = 8080, weight = 100 },
  ]
}

output "upstream_id" {
  value = apisix_upstream.basic.id
}
