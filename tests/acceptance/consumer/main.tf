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

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
  timeout   = 30
}

# Basic consumer
resource "apisix_consumer" "basic" {
  username = "test-consumer-basic"
  desc     = "Basic consumer for testing"
}

# Consumer with key-auth plugin
resource "apisix_consumer" "key_auth" {
  username = "test-consumer-key-auth"
  desc     = "Consumer with key-auth plugin"

  plugins = {
    "key-auth" = jsonencode({
      key = "test-key-12345"
    })
  }
}

# Consumer with jwt-auth plugin
resource "apisix_consumer" "jwt_auth" {
  username = "test-consumer-jwt-auth"
  desc     = "Consumer with jwt-auth plugin"

  plugins = {
    "jwt-auth" = jsonencode({
      key       = "jwt-test-key"
      secret    = "my-secret-key-12345678"
      algorithm = "HS256"
    })
  }
}

# Consumer with labels
resource "apisix_consumer" "with_labels" {
  username = "test-consumer-labels"
  desc     = "Consumer with labels"

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}

# Consumer with group_id (requires consumer group)
resource "apisix_consumer_group" "test_group" {
  group_id = "test-consumer-group"
  desc     = "Test consumer group for consumer testing"

  plugins = {
    "limit-count" = jsonencode({
      count         = 100
      time_window   = 60
      rejected_code = 429
    })
  }
}

resource "apisix_consumer" "with_group" {
  username = "test-consumer-with-group"
  desc     = "Consumer with group_id"
  group_id = apisix_consumer_group.test_group.group_id

  plugins = {
    "key-auth" = jsonencode({
      key = "grouped-consumer-key"
    })
  }
}

# Consumer with multiple auth plugins (hmac-auth)
resource "apisix_consumer" "hmac_auth" {
  username = "test-consumer-hmac-auth"
  desc     = "Consumer with hmac-auth plugin"

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
