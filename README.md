# Terraform Provider for Apache APISIX

[![Unit Tests](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/unit-tests.yml/badge.svg)](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/unit-tests.yml)
[![Acceptance Tests](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/acceptance-tests.yml/badge.svg)](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/acceptance-tests.yml)
[![Release](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/release.yml/badge.svg)](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/release.yml)

A Terraform/OpenTofu provider for managing [Apache APISIX](https://apisix.apache.org/) API Gateway configurations.

## Features

### Resources

This provider supports the following APISIX resources:

- **`apisix_upstream`** - Manage upstream servers and load balancing configurations
  - Round-robin, least connections, consistent hashing
  - Health checks and passive health monitoring
  - Active health checks with custom parameters
  - TLS configuration for upstream connections

- **`apisix_route`** - Configure API routes and traffic routing rules
  - URI, host, method, and IP-based matching
  - Custom variables and filtering
  - Plugin configurations per route
  - WebSocket support
  - Custom Lua scripts

- **`apisix_service`** - Define reusable service configurations
  - Plugin chains for services
  - Host-based routing
  - WebSocket support
  - Custom Lua scripts

- **`apisix_consumer`** - Manage API consumers and authentication
  - Key authentication (key-auth)
  - JWT authentication (jwt-auth)
  - HMAC authentication (hmac-auth)
  - Basic authentication (basic-auth)
  - Consumer groups support

- **`apisix_consumer_group`** - Group consumers with shared policies
  - Reusable plugin configurations
  - Rate limiting per group
  - Access control policies

- **`apisix_plugin_config`** - Create reusable plugin configurations
  - DRY plugin management
  - Reference from multiple routes
  - Centralized plugin updates

- **`apisix_global_rule`** - Apply global plugins to all requests
  - System-wide rate limiting
  - Global CORS policies
  - IP restrictions
  - Logging and tracing

- **`apisix_ssl`** - Manage SSL/TLS certificates
  - Multiple SNI support
  - mTLS (mutual TLS) configuration
  - TLS version control
  - Certificate rotation

## Requirements

- **Terraform/OpenTofu**: >= 1.0
- **Go**: >= 1.21 (for building from source)
- **Apache APISIX**: >= 3.0

## Installation

### From OpenTofu Registry (Recommended)

```hcl
terraform {
  required_providers {
    apisix = {
      source  = "scicore/apisix"
      version = ">= 0.1.0"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "your-admin-key"
}
```

### From Source

```bash
git clone https://github.com/scicore-unibas-ch/terraform-provider-apisix.git
cd terraform-provider-apisix
make build
```

## Example Usage

### Basic Upstream and Route

```hcl
provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

resource "apisix_upstream" "backend" {
  name = "backend-service"
  type = "roundrobin"

  nodes {
    host   = "10.0.1.10"
    port   = 8080
    weight = 100
  }

  nodes {
    host   = "10.0.1.11"
    port   = 8080
    weight = 50
  }
}

resource "apisix_route" "api" {
  name = "api-route"
  uri  = "/api/*"

  upstream_id = apisix_upstream.backend.id

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
    })
  }
}
```

### Consumer with Authentication

```hcl
resource "apisix_consumer" "api_user" {
  username = "api-user-1"

  plugins = {
    "key-auth" = jsonencode({
      key = "secret-api-key-12345"
    })
  }
}

resource "apisix_route" "protected" {
  name = "protected-route"
  uri  = "/protected/*"

  upstream_id = apisix_upstream.backend.id

  plugins = {
    "key-auth" = jsonencode({})
  }
}
```

### Global Rate Limiting

```hcl
resource "apisix_global_rule" "global_rate_limit" {
  rule_id = "global-rate-limit"

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

## Documentation

Full documentation is available in the [`docs/`](docs/) directory:

- [Upstream Resource](docs/resources/upstream.md)
- [Route Resource](docs/resources/route.md)
- [Service Resource](docs/resources/service.md)
- [Consumer Resource](docs/resources/consumer.md)
- [Consumer Group Resource](docs/resources/consumer_group.md)
- [Plugin Config Resource](docs/resources/plugin_config.md)
- [Global Rule Resource](docs/resources/global_rule.md)
- [SSL Resource](docs/resources/ssl.md)

## Development

### Building

```bash
make build
```

### Running Tests

```bash
# Unit tests
make test

# Acceptance tests (requires Docker Compose)
make test-env-up        # Start APISIX cluster
make test-acceptance    # Run all acceptance tests
make test-env-down      # Stop cluster

# Run specific acceptance test
make test-acceptance-single TEST=upstream
make test-acceptance-single TEST=route
make test-acceptance-single TEST=service
```

### Test Coverage

The provider has comprehensive test coverage:

- **Unit Tests**: 77 tests covering all resources
- **Acceptance Tests**: 44 tests with real APISIX instances
- **Total**: 121 tests with 100% pass rate

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Apache APISIX](https://apisix.apache.org/) - The amazing API gateway
- [OpenTofu](https://opentofu.org/) - Open source infrastructure as code
- [Terraform](https://www.terraform.io/) - Original infrastructure as code tool
