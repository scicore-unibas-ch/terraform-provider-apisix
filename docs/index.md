# APISIX Provider

The APISIX Provider is used to interact with Apache APISIX API Gateway resources.

## Example Usage

```hcl
terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "0.1.0"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

resource "apisix_upstream" "example" {
  name = "example-upstream"
  type = "roundrobin"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }
}
```

## Authentication and Configuration

The APISIX Provider requires the following configuration:

- `base_url` - (Required) The base URL of the APISIX Admin API. Can be set via the `APISIX_BASE_URL` environment variable.
- `admin_key` - (Required) The API key for authenticating with the APISIX Admin API. Can be set via the `APISIX_ADMIN_KEY` environment variable.

### Using Environment Variables

```hcl
provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
}
```

```bash
export APISIX_BASE_URL="http://localhost:9180/apisix/admin"
export APISIX_ADMIN_KEY="your-admin-key"
```

## Supported Resources

- [apisix_upstream](resources/upstream.md) - Manages an APISIX Upstream
- [apisix_route](resources/route.md) - Manages an APISIX Route
- [apisix_service](resources/service.md) - Manages an APISIX Service
- [apisix_consumer](resources/consumer.md) - Manages an APISIX Consumer
- [apisix_consumer_group](resources/consumer_group.md) - Manages an APISIX Consumer Group
- [apisix_plugin_config](resources/plugin_config.md) - Manages an APISIX Plugin Config
- [apisix_global_rule](resources/global_rule.md) - Manages an APISIX Global Rule
- [apisix_ssl](resources/ssl.md) - Manages an APISIX SSL Certificate

## Requirements

- Apache APISIX 3.13.0 or later
- Terraform 1.0 or later
- OpenTofu 1.6 or later
