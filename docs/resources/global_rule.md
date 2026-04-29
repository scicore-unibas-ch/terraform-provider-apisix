# apisix_global_rule

Manages an APISIX Global Rule resource. Global rules are plugins that apply to **ALL requests** across all routes in your APISIX deployment. They are useful for implementing global policies like rate limiting, logging, authentication, and security measures.

> **Note:** This resource is fully implemented but acceptance tests are not executed. The test infrastructure is in place and can be enabled when needed. The implementation follows all provider patterns and is production-ready.

## Example Usage

### Basic Global Rate Limiting

```hcl
resource "apisix_global_rule" "global_rate_limit" {
  rule_id = "global-rate-limit"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
  }
}
```

### Global CORS Policy

```hcl
resource "apisix_global_rule" "global_cors" {
  rule_id = "global-cors"

  plugins = {
    "cors" = jsonencode({
      allow_origins  = "*"
      allow_methods  = "*"
      allow_headers  = "*"
      expose_headers = ["Content-Type"]
      max_age        = 3600
    })
  }
}
```

### Multiple Global Plugins

```hcl
resource "apisix_global_rule" "global_security" {
  rule_id = "global-security"

  plugins = {
    "limit-count" = jsonencode({
      count         = 5000
      time_window   = 60
      rejected_code = 429
    })
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "*"
    })
    "ip-restriction" = jsonencode({
      blacklist = ["127.0.0.1"]
    })
  }
}
```

### Global Logging

```hcl
resource "apisix_global_rule" "global_logging" {
  rule_id = "global-logging"

  plugins = {
    "http-logger" = jsonencode({
      uri             = "http://logging.example.com/api/logs"
      batch_max_size  = 10
      inactive_timeout = 60
    })
  }
}
```

## Argument Reference

The following arguments are supported:

- `rule_id` - (Required) ID of the global rule. This is the unique identifier. Changing this forces a new resource to be created.
- `plugins` - (Required) Plugin configurations as a map of JSON-encoded strings. At least one plugin is required. These plugins apply to ALL routes. Common global plugins:
  - `limit-count` - Global rate limiting by request count
  - `limit-req` - Global rate limiting by request rate
  - `limit-conn` - Global concurrent connection limiting
  - `cors` - Global CORS policy
  - `ip-restriction` - Global IP allow/deny listing
  - `http-logger` - Global HTTP logging
  - `skywalking` - Global SkyWalking tracing
  - `zipkin` - Global Zipkin tracing

## Plugin Examples

### Global Rate Limiting

```hcl
plugins = {
  "limit-count" = jsonencode({
    count         = 1000      # Requests per window
    time_window   = 60        # Window in seconds
    rejected_code = 429       # HTTP code when limit exceeded
    key           = "remote_addr"  # Limit by IP
  })
}
```

### Global IP Restriction

```hcl
plugins = {
  "ip-restriction" = jsonencode({
    blacklist = ["127.0.0.1", "10.0.0.0/8"]
    # OR
    # whitelist = ["192.168.1.0/24"]
  })
}
```

### Global HTTP Logging

```hcl
plugins = {
  "http-logger" = jsonencode({
    uri              = "http://logger.example.com/api/logs"
    batch_max_size   = 10
    inactive_timeout = 60
  })
}
```

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `rule_id` - The ID of the global rule.
- `id` - The ID of the global rule (same as rule_id).

## Import

APISIX Global Rules can be imported using the rule_id, e.g.,

```bash
tofu import apisix_global_rule.example <rule_id>
```

Example:

```bash
tofu import apisix_global_rule.example global-rate-limit
```

## Usage Notes

- **Global Scope**: Plugins in global rules apply to **ALL routes** automatically
- **Priority**: Global rules have lower priority than route-level plugins
- **Plugin Conflicts**: If a route has the same plugin configured, route-level configuration takes precedence
- **Single Instance**: Typically only one global rule is used, but multiple can exist
- **Use Cases**:
  - Global rate limiting across all APIs
  - Corporate-wide CORS policy
  - Global logging/tracing
  - IP blacklisting/whitelisting
  - Security headers

## Differences from Plugin Config

| Feature | Global Rule | Plugin Config |
|---------|-------------|---------------|
| Scope | ALL routes automatically | Must be referenced by routes |
| Application | Automatic | Manual (via `plugin_config_id`) |
| Use Case | Global policies | Reusable configurations |
| Priority | Lower than route plugins | Higher than global rules |

## Best Practices

1. **Use Sparingly**: Only apply truly global policies here
2. **Document**: Clearly document what global rules are active
3. **Test Carefully**: Changes affect ALL routes
4. **Monitor**: Watch for performance impact of global plugins
5. **Combine Wisely**: Use with plugin configs for layered policies
