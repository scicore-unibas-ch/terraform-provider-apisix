---
page_title: "apisix_plugin_config Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages an APISIX Plugin Config — a reusable bundle of plugins that routes can reference via plugin_config_id.
---

# apisix_plugin_config

Manages an APISIX [Plugin Config](https://apisix.apache.org/docs/apisix/terminology/plugin-config/) — a reusable, named bundle of plugin configurations. Routes reference a plugin config by its `id` via the `plugin_config_id` attribute, which avoids duplicating plugin definitions across many routes.

APISIX requires the `plugins` field on a plugin config, but it may be empty (`plugins = {}`).

## Example Usage

```hcl
resource "apisix_plugin_config" "standard_api" {
  id   = "standard-api"
  desc = "Standard rate limiting + CORS for public API routes"

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
    "cors" = jsonencode({
      allow_origins = "*"
      allow_methods = "GET,POST"
    })
  }

  labels = {
    env  = "production"
    team = "platform"
  }
}

resource "apisix_route" "users" {
  id               = "users-api"
  uri              = "/users/*"
  upstream_id      = apisix_upstream.users.id
  plugin_config_id = apisix_plugin_config.standard_api.id
}
```

## Argument Reference

- `id` — (Required, ForceNew) Unique identifier for the plugin config. Changing this forces replacement.
- `plugins` — (Required) Map of plugin name to JSON-encoded configuration. Required, but may be empty (`plugins = {}`): APISIX requires the field to be present, not to hold any plugins. JSON-equivalent values are suppressed by the provider.
- `desc` — (Optional) Description.
- `labels` — (Optional) Map of string key/value pairs.

## Import

```bash
terraform import apisix_plugin_config.standard_api standard-api
```
