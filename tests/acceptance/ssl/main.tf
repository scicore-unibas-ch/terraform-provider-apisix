terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
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

locals {
  cert_dir = "${path.module}/certs"
}

# Basic SSL certificate
resource "apisix_ssl" "basic" {
  sni   = "example.com"
  cert  = file("${local.cert_dir}/example.com.crt")
  key   = file("${local.cert_dir}/example.com.key")
}

# SSL certificate for secure.example.com
resource "apisix_ssl" "secure" {
  sni   = "secure.example.com"
  cert  = file("${local.cert_dir}/secure.example.com.crt")
  key   = file("${local.cert_dir}/secure.example.com.key")
}

# SSL certificate with labels
resource "apisix_ssl" "with_labels" {
  sni   = "labeled.example.com"
  cert  = file("${local.cert_dir}/labeled.example.com.crt")
  key   = file("${local.cert_dir}/labeled.example.com.key")
  labels = {
    env        = "production"
    managed-by = "terraform"
    team       = "platform"
  }
}

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
}
