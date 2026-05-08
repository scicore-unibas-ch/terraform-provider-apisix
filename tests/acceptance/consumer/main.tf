terraform {
  required_providers {
    apisix = {
      source = "scicore-unibas-ch/apisix"
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

resource "apisix_consumer" "basic" {
  id   = "test-consumer-basic"
  desc = "Basic consumer for testing"
}

resource "apisix_consumer" "key_auth" {
  id   = "test-consumer-key-auth"
  desc = "Consumer with key-auth plugin"

  plugins = {
    "key-auth" = jsonencode({
      key = "test-key-12345"
    })
  }
}

resource "apisix_consumer" "jwt_auth" {
  id   = "test-consumer-jwt-auth"
  desc = "Consumer with jwt-auth plugin"

  plugins = {
    "jwt-auth" = jsonencode({
      key       = "jwt-test-key"
      secret    = "my-secret-key-12345678"
      algorithm = "HS256"
    })
  }
}

resource "apisix_consumer" "with_labels" {
  id   = "test-consumer-labels"
  desc = "Consumer with labels"

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}

resource "apisix_consumer_group" "test_group" {
  id   = "test-consumer-group"
  desc = "Test consumer group for consumer testing"

  plugins = {
    "limit-count" = jsonencode({
      count         = 100
      time_window   = 60
      rejected_code = 429
    })
  }
}

resource "apisix_consumer" "with_group" {
  id       = "test-consumer-with-group"
  desc     = "Consumer with group_id"
  group_id = apisix_consumer_group.test_group.id

  plugins = {
    "key-auth" = jsonencode({
      key = "grouped-consumer-key"
    })
  }
}

resource "apisix_consumer" "hmac_auth" {
  id   = "test-consumer-hmac-auth"
  desc = "Consumer with hmac-auth plugin"

  plugins = {
    "hmac-auth" = jsonencode({
      key_id         = "hmac-key-id"
      secret_key     = "hmac-secret-key-12345678"
      algorithm      = "hmac-sha512"
      clock_skew     = 300
      keep_headers   = "false"
      encoded_header = "false"
    })
  }
}
