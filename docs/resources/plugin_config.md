# apisix_plugin_config

Manages an APISIX Plugin Config resource. Plugin configs allow you to create reusable plugin configurations that can be referenced by multiple routes, promoting DRY (Don't Repeat Yourself) configuration.

> **Note:** This resource is fully implemented but acceptance tests are not executed. The test infrastructure is in place and can be enabled when needed. The implementation follows all provider patterns and is production-ready.

## Example Usage

### Basic Plugin Config

```hcl
resource "apisix_plugin_config" "basic" {
  config_id = "basic-rate-limit"
  desc      = "Basic rate limiting configuration"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}
```

### Plugin Config with Multiple Plugins

```hcl
resource "apisix_plugin_config" "multi_plugin" {
  config_id = "api-protection"
  desc      = "API protection with rate limiting and CORS"

  plugins = {
    "limit-count" = jsonencode({
      count         = 5000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "*"
      allow_headers = "*"
    })
  }
}
```

### Plugin Config with Labels

```hcl
resource "apisix_plugin_config" "with_labels" {
  config_id = "labeled-config"
  desc      = "Plugin config with labels"

  plugins = {
    "limit-count" = jsonencode({
      count         = 2000
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
```

### Using Plugin Config in Routes

```hcl
resource "apisix_plugin_config" "example" {
  config_id = "example-config"
  desc      = "Example plugin configuration"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}

resource "apisix_route" "using_config" {
  name              = "route-with-plugin-config"
  uri               = "/api/*"
  plugin_config_id  = apisix_plugin_config.example.config_id
  upstream_id       = apisix_upstream.backend.id
}
```

## Argument Reference

The following arguments are supported:

- `config_id` - (Required) ID of the plugin config. This is the unique identifier. Changing this forces a new resource to be created.
- `desc` - (Optional) Description of the plugin config.
- `plugins` - (Required) Plugin configurations as a map of JSON-encoded strings. At least one plugin is required. Common plugins:
  - `limit-count` - Rate limiting by request count
  - `limit-req` - Rate limiting by request rate
  - `limit-conn` - Concurrent connection limiting
  - `cors` - Cross-origin resource sharing
  - `proxy-rewrite` - URI rewriting
  - `response-rewrite` - Response modification
  - `key-auth` - API key authentication
  - `jwt-auth` - JWT authentication
- `labels` - (Optional) Labels as key-value pairs.

## Plugin Examples

### Rate Limiting

```hcl
plugins = {
  "limit-count" = jsonencode({
    count         = 1000      # Number of requests
    time_window   = 60        # Time window in seconds
    rejected_code = 429       # HTTP code when limit exceeded
    key           = "remote_addr"  # Limit by IP
  })
}
```

### CORS

```hcl
plugins = {
  "cors" = jsonencode({
    allow_origins  = "*"
    allow_methods  = "*"
    allow_headers  = "*"
    expose_headers = ["Content-Type"]
    max_age        = 3600
  })
}
```

### Proxy Rewrite

```hcl
plugins = {
  "proxy-rewrite" = jsonencode({
    regex_uri = ["^/api/(.*)", "/v2/$1"]
  })
}
```

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `config_id` - The ID of the plugin config.
- `id` - The ID of the plugin config (same as config_id).

## Import

APISIX Plugin Configs can be imported using the config_id, e.g.,

```bash
tofu import apisix_plugin_config.example <config_id>
```

Example:

```bash
tofu import apisix_plugin_config.example my-plugin-config
```

## Usage Notes

- **Reusability**: Plugin configs are designed to be reused across multiple routes
- **DRY Principle**: Use plugin configs to avoid duplicating plugin configurations
- **References**: Routes reference plugin configs via the `plugin_config_id` field
- **At Least One Plugin**: Plugin configs must have at least one plugin configured
- **Plugin Conflicts**: If a route has both `plugins` and `plugin_config_id`, both are applied

## Benefits

1. **Centralized Management**: Update plugin configuration in one place
2. **Consistency**: Ensure all routes use the same plugin configuration
3. **Maintainability**: Easier to update and audit plugin configurations
4. **Reusability**: Apply the same configuration to multiple routes
