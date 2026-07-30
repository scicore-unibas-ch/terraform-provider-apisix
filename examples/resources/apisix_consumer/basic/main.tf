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

resource "apisix_consumer" "basic" {
  id   = "basic-user"
  desc = "Basic consumer for API access"

  plugins = {
    "key-auth" = jsonencode({
      key = "basic-user-key"
    })
  }
}
