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
      key            = "hmac-user-key"
      secret         = "hmac-secret-key-12345678"
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
