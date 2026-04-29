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

# Basic consumer group
resource "apisix_consumer_group" "basic" {
  group_id = "test-group-basic"
  desc     = "Basic consumer group for testing"

  # Consumer groups require at least one plugin
  plugins = {
    "limit-count" = jsonencode({
      count         = 10000
      time_window   = 60
      rejected_code = 429
    })
  }
}

# Consumer group with plugins
resource "apisix_consumer_group" "with_plugins" {
  group_id = "test-group-with-plugins"
  desc     = "Consumer group with rate limiting plugins"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
  }
}

# Consumer group with multiple plugins
resource "apisix_consumer_group" "multi_plugins" {
  group_id = "test-group-multi-plugins"
  desc     = "Consumer group with multiple plugins"

  plugins = {
    "limit-count" = jsonencode({
      count         = 500
      time_window   = 60
      rejected_code = 503
    })
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "*"
      allow_headers = "*"
    })
  }
}

# Consumer group with name
resource "apisix_consumer_group" "with_name" {
  group_id = "test-group-with-name"
  name     = "Premium Tier Group"
  desc     = "Consumer group with name field"

  plugins = {
    "limit-count" = jsonencode({
      count         = 5000
      time_window   = 60
      rejected_code = 429
    })
  }
}

# Consumer group with labels
resource "apisix_consumer_group" "with_labels" {
  group_id = "test-group-labels"
  desc     = "Consumer group with labels"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}

# Consumer group for consumer testing
resource "apisix_consumer_group" "consumer_test" {
  group_id = "test-group-consumer"
  desc     = "Consumer group for testing consumer group_id"

  plugins = {
    "limit-count" = jsonencode({
      count         = 100
      time_window   = 60
      rejected_code = 429
    })
  }
}
