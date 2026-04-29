terraform {
  required_providers {
    apisix = {
      source  = "scicore/apisix"
      version = "0.1.0"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

# Basic SSL certificate
# Note: Requires SSL proxy enabled in APISIX
resource "apisix_ssl" "basic" {
  sni  = "example.com"
  cert = file("${path.module}/example.com.crt")
  key  = file("${path.module}/example.com.key")
}
