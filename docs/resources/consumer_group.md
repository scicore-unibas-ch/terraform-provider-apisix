---
page_title: "apisix_consumer_group Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages an APISIX Consumer Group. Plugins defined here apply to every consumer in the group.
---

# apisix_consumer_group

Manages an APISIX [Consumer Group](https://apisix.apache.org/docs/apisix/terminology/consumer-group/). Plugins attached to a consumer group apply to every consumer that references the group via `group_id`.

APISIX requires at least one plugin on a consumer group.

## Example Usage

```hcl
resource "apisix_consumer_group" "premium" {
  id   = "premium-tier"
  name = "Premium Tier"
  desc = "Higher rate limits for paying customers"

  plugins = {
    "limit-count" = jsonencode({
      count         = 10000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
  }

  labels = {
    env  = "production"
    tier = "premium"
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

## Argument Reference

- `id` — (Required, ForceNew) Unique identifier for the consumer group. Used as the APISIX object key. Changing this forces replacement.
- `plugins` — (Required) Map of plugin name to JSON-encoded plugin configuration. APISIX requires at least one plugin. JSON-equivalent values are suppressed by the provider so server-side normalization (key reordering, default injection) does not produce drift.
- `name` — (Optional) Human-readable name.
- `desc` — (Optional) Description.
- `labels` — (Optional) Map of string key/value pairs.

## Import

Consumer groups are imported by their `id`:

```bash
terraform import apisix_consumer_group.premium premium-tier
```
