terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "0.1.0"
    }
  }
}

# SSL Certificate Examples
# Note: These resources are fully implemented but acceptance tests are not executed.
#       Test infrastructure is available and can be enabled when SSL testing is required.

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

# SSL certificate with multiple SNIs and mTLS
resource "apisix_ssl" "advanced" {
  snis = ["api.example.com", "www.example.com"]
  cert = file("${path.module}/example.com.crt")
  key  = file("${path.module}/example.com.key")

  ssl_protocols = ["TLSv1.2", "TLSv1.3"]

  # Enable client certificate verification (mTLS)
  client {
    ca_cert = file("${path.module}/ca.crt")
    depth   = 2
  }

  labels = {
    env        = "production"
    team       = "security"
    managed-by = "terraform"
  }
}

# SSL certificate with TLS 1.3 only
resource "apisix_ssl" "tls13_only" {
  sni  = "secure.example.com"
  cert = file("${path.module}/secure.crt")
  key  = file("${path.module}/secure.key")

  ssl_protocols = ["TLSv1.3"]

  labels = {
    env        = "production"
    security   = "high"
    managed-by = "terraform"
  }
}
