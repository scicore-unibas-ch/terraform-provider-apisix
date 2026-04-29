terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "0.1.0"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

# Consumer with JWT authentication
resource "apisix_consumer" "jwt_user" {
  username = "jwt-user"
  desc     = "User with JWT authentication"

  plugins = {
    "jwt-auth" = jsonencode({
      key       = "jwt-user-key"
      secret    = "my-jwt-secret-key-12345678"
      algorithm = "HS256"
    })
  }

  labels = {
    env        = "production"
    auth-type  = "jwt"
    managed-by = "terraform"
  }
}

# Consumer with API key authentication
resource "apisix_consumer" "api_key_user" {
  username = "api-key-user"
  desc     = "User with API key authentication"

  plugins = {
    "key-auth" = jsonencode({
      key = "api-key-12345"
    })
  }

  labels = {
    env        = "production"
    auth-type  = "apikey"
    managed-by = "terraform"
  }
}

# Consumer with HMAC authentication
resource "apisix_consumer" "hmac_user" {
  username = "hmac-user"
  desc     = "User with HMAC authentication"

  plugins = {
    "hmac-auth" = jsonencode({
      key_id         = "hmac-user-key"
      secret_key     = "hmac-secret-key-12345678"
      algorithm      = "hmac-sha512"
      clock_skew     = 300
      keep_headers   = "false"
      encoded_header = "false"
    })
  }

  labels = {
    env        = "production"
    auth-type  = "hmac"
    managed-by = "terraform"
  }
}

# Consumer in a consumer group
resource "apisix_consumer_group" "example_group" {
  group_id = "example-consumer-group"
  name     = "Example Consumer Group"
  desc     = "Consumer group for example users"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }

  labels = {
    env        = "production"
    managed-by = "terraform"
  }
}

resource "apisix_consumer" "grouped_user" {
  username = "grouped-user"
  desc     = "User in a consumer group"
  group_id = apisix_consumer_group.example_group.group_id

  plugins = {
    "key-auth" = jsonencode({
      key = "grouped-user-key"
    })
  }

  labels = {
    env        = "production"
    user-type  = "grouped"
    managed-by = "terraform"
  }
}
