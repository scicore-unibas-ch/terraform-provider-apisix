# apisix_consumer

Manages an APISIX Consumer resource. Consumers are used for authentication and authorization, allowing you to manage API users with various authentication plugins.

## Example Usage

### Basic Consumer

```hcl
resource "apisix_consumer" "basic" {
  username = "basic-user"
  desc     = "Basic consumer without authentication"
}
```

### Consumer with Key Auth

```hcl
resource "apisix_consumer" "key_auth" {
  username = "key-auth-user"
  desc     = "Consumer with key-auth plugin"

  plugins = {
    "key-auth" = jsonencode({
      key = "my-secret-key"
    })
  }
}
```

### Consumer with JWT Auth

```hcl
resource "apisix_consumer" "jwt_auth" {
  username = "jwt-auth-user"
  desc     = "Consumer with jwt-auth plugin"

  plugins = {
    "jwt-auth" = jsonencode({
      key       = "jwt-key"
      secret    = "my-secret-key-12345678"
      algorithm = "HS256"
    })
  }
}
```

### Consumer with HMAC Auth

```hcl
resource "apisix_consumer" "hmac_auth" {
  username = "hmac-auth-user"
  desc     = "Consumer with hmac-auth plugin"

  plugins = {
    "hmac-auth" = jsonencode({
      key            = "hmac-key"
      secret         = "hmac-secret-key-12345678"
      algorithm      = "hmac-sha512"
      clock_skew     = 300
      keep_headers   = "false"
      encoded_header = "false"
    })
  }
}
```

### Consumer with Basic Auth

```hcl
resource "apisix_consumer" "basic_auth" {
  username = "basic-auth-user"
  desc     = "Consumer with basic-auth plugin"

  plugins = {
    "basic-auth" = jsonencode({
      username = "apiuser"
      password = "apipassword123"
    })
  }
}
```

### Consumer with Labels

```hcl
resource "apisix_consumer" "with_labels" {
  username = "labeled-user"
  desc     = "Consumer with labels"

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}
```

### Complete Consumer with All Fields

```hcl
resource "apisix_consumer" "complete" {
  username = "complete-user"
  desc     = "Complete consumer configuration"

  # Use key-auth for simple API key authentication
  plugins = {
    "key-auth" = jsonencode({
      key = "complete-user-key-12345"
    })
  }

  labels = {
    env        = "production"
    team       = "backend"
    managed-by = "terraform"
  }
}
```

## Argument Reference

The following arguments are supported:

- `username` - (Required) Username of the consumer. This is the unique identifier. Changing this forces a new resource to be created.
- `group_id` - (Optional) Group ID of the consumer. Requires a pre-existing consumer group with matching ID.
- `desc` - (Optional) Description of the consumer.
- `plugins` - (Optional) Plugin configurations as a map of JSON-encoded strings. Common authentication plugins:
  - `key-auth` - Simple API key authentication
  - `jwt-auth` - JWT token authentication
  - `hmac-auth` - HMAC signature authentication
  - `basic-auth` - HTTP Basic authentication
  - `wolf-rbac` - Wolf RBAC authentication
  - `openid-connect` - OpenID Connect authentication
- `labels` - (Optional) Labels as key-value pairs.

## Authentication Plugin Examples

### Key Auth Plugin

```hcl
plugins = {
  "key-auth" = jsonencode({
    key = "your-api-key"
  })
}
```

### JWT Auth Plugin

```hcl
plugins = {
  "jwt-auth" = jsonencode({
    key       = "jwt-key"
    secret    = "your-secret-key-min-16-chars"
    algorithm = "HS256"  # Options: HS256, HS512, RS256, RS512, ES256, ES512
    exp       = 86400    # Optional: token expiration in seconds
  })
}
```

### HMAC Auth Plugin

```hcl
plugins = {
  "hmac-auth" = jsonencode({
    key_id         = "hmac-key-id"    # Required: unique key identifier
    secret_key     = "your-hmac-secret"  # Required: secret key
    algorithm      = "hmac-sha512"    # Options: hmac-sha1, hmac-sha256, hmac-sha512
    clock_skew     = 300              # Optional: clock skew in seconds
    keep_headers   = "false"          # Optional: keep headers in request
    encoded_header = "false"          # Optional: use encoded header
  })
}
```

### Basic Auth Plugin

```hcl
plugins = {
  "basic-auth" = jsonencode({
    username = "apiuser"
    password = "secure-password"
  })
}
```

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `username` - The username of the consumer.
- `id` - The ID of the consumer (same as username).

## Import

APISIX Consumers can be imported using the username, e.g.,

```bash
tofu import apisix_consumer.example <username>
```

Example:

```bash
tofu import apisix_consumer.example test-user-1
```

## Usage with Routes

Consumers are typically used with routes that have authentication plugins enabled:

```hcl
resource "apisix_route" "protected" {
  name = "protected-route"
  uri  = "/api/*"

  upstream_id = apisix_upstream.backend.id

  plugins = {
    "key-auth" = jsonencode({})
  }
}
```

Clients must then include the API key in their requests:

```bash
curl -H "apikey: your-api-key" http://apisix:9080/api/endpoint
```
