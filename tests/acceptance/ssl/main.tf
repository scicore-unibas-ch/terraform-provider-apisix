terraform {
  required_providers {
    apisix = {
      source  = "scicore/apisix"
      version = "0.1.0"
    }
  }
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

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
  timeout   = 30
}

# Basic SSL certificate
resource "apisix_ssl" "basic" {
  sni  = "example.com"
  cert = file("${path.module}/example.com.crt")
  key  = file("${path.module}/example.com.key")
}

# SSL with multiple SNIs
resource "apisix_ssl" "multi_sni" {
  snis = ["api.example.com", "www.example.com"]
  cert = file("${path.module}/example.com.crt")
  key  = file("${path.module}/example.com.key")

  ssl_protocols = ["TLSv1.2", "TLSv1.3"]
}

# SSL with TLS 1.3 only
resource "apisix_ssl" "tls13" {
  sni  = "secure.example.com"
  cert = file("${path.module}/example.com.crt")
  key  = file("${path.module}/example.com.key")

  ssl_protocols = ["TLSv1.3"]
}

# SSL with labels
resource "apisix_ssl" "with_labels" {
  sni  = "labeled.example.com"
  cert = file("${path.module}/example.com.crt")
  key  = file("${path.module}/example.com.key")

  ssl_protocols = ["TLSv1.2", "TLSv1.3"]

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}
