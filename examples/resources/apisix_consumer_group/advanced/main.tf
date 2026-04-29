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

# Consumer group with rate limiting
resource "apisix_consumer_group" "rate_limited" {
  group_id = "rate-limited-group"
  desc     = "Consumer group with rate limiting"

  plugins = {
    "limit-count" = jsonencode({
      count         = 5000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
  }

  labels = {
    env        = "production"
    tier       = "premium"
    managed-by = "terraform"
  }
}

# Consumer in the group
resource "apisix_consumer" "premium_user" {
  username = "premium-user"
  desc     = "Premium user in rate-limited group"
  group_id = apisix_consumer_group.rate_limited.group_id

  plugins = {
    "key-auth" = jsonencode({
      key = "premium-user-key"
    })
  }

  labels = {
    env        = "production"
    user-type  = "premium"
    managed-by = "terraform"
  }
}
