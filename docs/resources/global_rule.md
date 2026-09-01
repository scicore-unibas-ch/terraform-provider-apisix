---
page_title: "apisix_global_rule Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages an APISIX Global Rule. Plugins on a global rule apply to every request flowing through APISIX, regardless of route.
---

# apisix_global_rule

Manages an APISIX [Global Rule](https://apisix.apache.org/docs/apisix/terminology/global-rule/). Plugins on a global rule apply to **every** request flowing through APISIX, regardless of route — typical use cases are global rate limiting, system-wide CORS policies, IP restrictions, and tracing.

APISIX requires the `plugins` field on a global rule, but it may be empty (`plugins = {}`).

## Example Usage

```hcl
resource "apisix_global_rule" "rate_limit" {
  id = "global-rate-limit"

  plugins = {
    "limit-count" = jsonencode({
      count         = 10000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
  }
}
```

### Multiple global plugins

```hcl
resource "apisix_global_rule" "security" {
  id = "global-security"

  plugins = {
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "GET,POST,PUT,DELETE"
    })
    "ip-restriction" = jsonencode({
      blacklist = ["192.168.0.0/16"]
    })
  }
}
```

## Argument Reference

- `id` — (Required, ForceNew) Unique identifier for the global rule. Changing this forces replacement.
- `plugins` — (Required) Map of plugin name to JSON-encoded configuration. Required, but may be empty (`plugins = {}`): APISIX requires the field to be present, not to hold any plugins. JSON-equivalent values are suppressed by the provider.

## Import

```bash
terraform import apisix_global_rule.rate_limit global-rate-limit
```
