---
page_title: "APISIX Provider"
subcategory: ""
description: |-
  Terraform/OpenTofu provider for managing Apache APISIX API Gateway resources via the Admin API.
---

# APISIX Provider

The APISIX provider lets you manage [Apache APISIX](https://apisix.apache.org/) API Gateway resources (upstreams, routes, services, consumers, plugins, global rules) through the APISIX Admin API.

The provider is built on the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) and is compatible with both Terraform and OpenTofu.

## Example Usage

```hcl
terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "~> 0.2"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "your-admin-key"
}

resource "apisix_upstream" "example" {
  id   = "example-upstream"
  type = "roundrobin"

  nodes = [
    { host = "127.0.0.1", port = 8080, weight = 100 },
  ]
}
```

## Schema

### Required

- `base_url` — Base URL of the APISIX Admin API (e.g. `http://localhost:9180/apisix/admin`). May also be set via the `APISIX_BASE_URL` environment variable.
- `admin_key` — Admin API key. Sensitive. May also be set via the `APISIX_ADMIN_KEY` environment variable.

### Optional

- `timeout` — HTTP client timeout in seconds. Defaults to `30`.
- `insecure` — Skip TLS certificate verification when talking to the Admin API. Defaults to `false`.

## Environment Variables

```bash
export APISIX_BASE_URL="http://localhost:9180/apisix/admin"
export APISIX_ADMIN_KEY="your-admin-key"
```

## Supported Resources

- [apisix_upstream](resources/upstream.md) — backend definition (nodes or service-discovery reference) used by routes and services.
- [apisix_route](resources/route.md) — request matching rule that maps incoming requests to a backend.
- [apisix_service](resources/service.md) — reusable bundle of host/plugin/upstream config.
- [apisix_consumer](resources/consumer.md) — authenticated API client identity.
- [apisix_consumer_group](resources/consumer_group.md) — group of consumers sharing plugin configuration.
- [apisix_plugin_config](resources/plugin_config.md) — reusable bundle of plugins for routes.
- [apisix_global_rule](resources/global_rule.md) — plugins applied to every request through APISIX.

## Requirements

- Apache APISIX 3.14.0 or later
- Terraform >= 1.0 or OpenTofu >= 1.6
