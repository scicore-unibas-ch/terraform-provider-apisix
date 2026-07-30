terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "~> 0.5"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

resource "apisix_upstream" "backend" {
  id   = "basic-service-backend"
  type = "roundrobin"

  nodes = [
    { host = "127.0.0.1", port = 8080, weight = 100 },
  ]
}

resource "apisix_service" "basic" {
  id   = "basic-service"
  name = "basic-service"
  desc = "Basic service for testing"

  upstream_id = apisix_upstream.backend.id
}
