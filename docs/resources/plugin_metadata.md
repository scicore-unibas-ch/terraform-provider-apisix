---
page_title: "apisix_plugin_metadata Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages APISIX Plugin Metadata. The metadata body shape is plugin-specific; consult the APISIX plugin documentation.
---

# apisix_plugin_metadata

Manages APISIX [Plugin Metadata](https://apisix.apache.org/docs/apisix/admin-api/#plugin-metadata). Plugin metadata is a global, per-plugin configuration applied to every instance of that plugin — typically log format strings, exporter endpoints, or default tracing parameters.

The body of `metadata` is plugin-specific: consult the documentation of the plugin you are configuring (e.g. [`syslog`](https://apisix.apache.org/docs/apisix/plugins/syslog/), [`http-logger`](https://apisix.apache.org/docs/apisix/plugins/http-logger/), [`prometheus`](https://apisix.apache.org/docs/apisix/plugins/prometheus/)) for the accepted shape.

## Example Usage

```hcl
resource "apisix_plugin_metadata" "syslog" {
  id = "syslog"
  metadata = jsonencode({
    log_format = {
      host         = "$host"
      "@timestamp" = "$time_iso8601"
      client_ip    = "$remote_addr"
    }
  })
}

resource "apisix_plugin_metadata" "http_logger" {
  id = "http-logger"
  metadata = jsonencode({
    log_format = {
      host         = "$host"
      "@timestamp" = "$time_iso8601"
      method       = "$request_method"
      uri          = "$request_uri"
    }
  })
}
```

## Argument Reference

- `id` — (Required, ForceNew) Name of the plugin (e.g. `syslog`, `http-logger`, `prometheus`). Used as the APISIX object key. Changing this forces replacement.
- `metadata` — (Required) JSON-encoded plugin metadata body. Must be a JSON object. JSON-equivalent values (key reordering, whitespace) are suppressed by the provider so server-side normalization does not produce drift.

The provider parses `metadata` at plan time; malformed JSON or a non-object root produces a plan-time error.

## Notes

- Some plugins (e.g. `prometheus`, `datadog`) inject defaults into the metadata body. If APISIX adds keys that are not present in your config, a no-op plan may still show a diff — include those keys explicitly in `jsonencode({...})` to suppress it, or open an issue to extend the suppression logic.
- Plugin metadata may contain sensitive material (Kafka SASL credentials, Syslog auth tokens). The `metadata` attribute is **not** marked `sensitive` because doing so would hide diffs entirely; treat it like any other config field and avoid committing secrets to VCS.

## Import

Plugin metadata is imported by the plugin name:

```bash
terraform import apisix_plugin_metadata.syslog syslog
```
