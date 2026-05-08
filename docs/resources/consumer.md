---
page_title: "apisix_consumer Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages an APISIX Consumer (an authenticated API client identity).
---

# apisix_consumer

Manages an APISIX [Consumer](https://apisix.apache.org/docs/apisix/terminology/consumer/). The Terraform `id` attribute is the consumer username (the URL key used by the APISIX Admin API).

## Example Usage

```hcl
resource "apisix_consumer" "alice" {
  id   = "alice"
  desc = "Alice's API client"

  plugins = {
    "key-auth" = jsonencode({
      key = "alice-secret-key"
    })
  }

  labels = {
    env  = "production"
    team = "platform"
  }
}
```

### Consumer attached to a Consumer Group

```hcl
resource "apisix_consumer_group" "premium" {
  id = "premium-tier"
  plugins = {
    "limit-count" = jsonencode({ count = 10000, time_window = 60, rejected_code = 429 })
  }
}

resource "apisix_consumer" "alice" {
  id       = "alice"
  group_id = apisix_consumer_group.premium.id

  plugins = {
    "key-auth" = jsonencode({ key = "alice-secret-key" })
  }
}
```

### Consumer with multiple authentication plugins

```hcl
resource "apisix_consumer" "service_account" {
  id = "svc-account"

  plugins = {
    "jwt-auth" = jsonencode({
      key       = "svc-jwt-key"
      secret    = "long-jwt-shared-secret"
      algorithm = "HS256"
    })
    "hmac-auth" = jsonencode({
      key_id     = "svc-hmac-id"
      secret_key = "long-hmac-shared-secret"
      algorithm  = "hmac-sha512"
    })
  }
}
```

## Argument Reference

- `id` — (Required, ForceNew) Consumer username. Used as the APISIX object key. Changing this forces replacement.
- `group_id` — (Optional) ID of an `apisix_consumer_group` this consumer belongs to.
- `desc` — (Optional) Description.
- `plugins` — (Optional) Map of plugin name to JSON-encoded configuration. Common values include `key-auth`, `jwt-auth`, `basic-auth`, `hmac-auth`. JSON-equivalent values are suppressed by the provider.
- `labels` — (Optional) Map of string key/value pairs.

## Import

Consumers are imported by their username (`id`):

```bash
terraform import apisix_consumer.alice alice
```
