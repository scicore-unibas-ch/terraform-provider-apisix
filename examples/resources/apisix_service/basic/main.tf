terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "0.1.0"
    }
  }
}

provider "apisix" {
  api_key = "test123456789"
}

resource "apisix_upstream" "backend" {
  name = "backend-upstream"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}

resource "apisix_service" "basic" {
  name = "basic-service"
  desc = "Basic service for testing"

  upstream_id = apisix_upstream.backend.id
}
