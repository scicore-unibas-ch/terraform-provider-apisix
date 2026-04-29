# apisix_consumer_group

Manages an APISIX Consumer Group resource. Consumer groups allow you to apply shared plugin configurations to multiple consumers, making it easier to manage common rate limits, authentication, and other policies across groups of users.

## Example Usage

### Basic Consumer Group

```hcl
resource "apisix_consumer_group" "basic" {
  group_id = "basic-group"
  desc     = "Basic consumer group"

  # Consumer groups require at least one plugin
  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}
```

### Consumer Group with Rate Limiting

```hcl
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
}
```

### Consumer Group with Multiple Plugins

```hcl
resource "apisix_consumer_group" "multi_plugin" {
  group_id = "multi-plugin-group"
  desc     = "Consumer group with multiple plugins"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
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
```

### Consumer Group with Labels

```hcl
resource "apisix_consumer_group" "with_labels" {
  group_id = "labeled-group"
  desc     = "Consumer group with labels"

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

### Consumer with Consumer Group

```hcl
resource "apisix_consumer_group" "example" {
  group_id = "example-group"
  desc     = "Example consumer group"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}

resource "apisix_consumer" "example" {
  username = "example-user"
  desc     = "User in example group"
  group_id = apisix_consumer_group.example.group_id

  plugins = {
    "key-auth" = jsonencode({
      key = "user-api-key"
    })
  }
}
```

## Argument Reference

The following arguments are supported:

- `group_id` - (Required) ID of the consumer group. This is the unique identifier. Changing this forces a new resource to be created.
- `desc` - (Optional) Description of the consumer group.
- `plugins` - (Required) Plugin configurations as a map of JSON-encoded strings. **Consumer groups require at least one plugin.** Common plugins:
  - `limit-count` - Rate limiting by count/time window
  - `limit-req` - Rate limiting by request rate
  - `limit-conn` - Concurrent connection limiting
  - `cors` - Cross-origin resource sharing
  - `ip-restriction` - IP allow/deny listing
  - `consumer-restriction` - Restrict by consumer name
- `labels` - (Optional) Labels as key-value pairs.

## Plugin Examples

### Rate Limiting Plugin

```hcl
plugins = {
  "limit-count" = jsonencode({
    count         = 1000      # Number of requests allowed
    time_window   = 60        # Time window in seconds
    rejected_code = 429       # HTTP code when limit exceeded
    key           = "remote_addr"  # Limit by IP address
  })
}
```

### CORS Plugin

```hcl
plugins = {
  "cors" = jsonencode({
    allow_origins = "*"
    allow_methods = "*"
    allow_headers = "*"
  })
}
```

### IP Restriction Plugin

```hcl
plugins = {
  "ip-restriction" = jsonencode({
    whitelist = ["127.0.0.1", "10.0.0.0/8"]
  })
}
```

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `group_id` - The ID of the consumer group.
- `id` - The ID of the consumer group (same as group_id).

## Import

APISIX Consumer Groups can be imported using the group_id, e.g.,

```bash
tofu import apisix_consumer_group.example <group_id>
```

Example:

```bash
tofu import apisix_consumer_group.example test-group-1
```

## Usage Notes

- **Plugins are required**: Unlike individual consumers, consumer groups must have at least one plugin configured.
- **Consumer membership**: Consumers reference consumer groups via the `group_id` field.
- **Plugin inheritance**: Consumers in a group inherit the group's plugin configurations in addition to their own plugins.
- **Use cases**: Consumer groups are ideal for applying common policies (rate limits, CORS, IP restrictions) to multiple consumers without duplicating configuration.
