# Terraform / OpenTofu Provider for Apache APISIX

[![Unit Tests](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/unit-tests.yml/badge.svg)](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/unit-tests.yml)
[![Acceptance Tests](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/acceptance-tests.yml/badge.svg)](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/acceptance-tests.yml)
[![Release](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/release.yml/badge.svg)](https://github.com/scicore-unibas-ch/terraform-provider-apisix/actions/workflows/release.yml)

Manage [Apache APISIX](https://apisix.apache.org/) API Gateway resources from Terraform or OpenTofu via the APISIX Admin API.

The provider is built on the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) (plugin protocol v6) and is verified against APISIX 3.14, 3.15, 3.16, and 3.17 in CI on every change.

## Resources

| Resource | Description |
| --- | --- |
| [`apisix_upstream`](docs/resources/upstream.md) | Backend definition: nodes (or service-discovery), load balancing, timeouts, keepalive, mTLS, health checks. |
| [`apisix_route`](docs/resources/route.md) | Request matching rule that maps incoming requests to a backend. |
| [`apisix_service`](docs/resources/service.md) | Reusable bundle of host/plugin/upstream config that routes can reference. |
| [`apisix_consumer`](docs/resources/consumer.md) | Authenticated API client identity (key-auth, jwt-auth, hmac-auth, basic-auth). |
| [`apisix_consumer_group`](docs/resources/consumer_group.md) | Group of consumers sharing plugin configuration. |
| [`apisix_plugin_config`](docs/resources/plugin_config.md) | Reusable bundle of plugins for routes. |
| [`apisix_global_rule`](docs/resources/global_rule.md) | Plugins applied to every request through APISIX. |
| [`apisix_plugin_metadata`](docs/resources/plugin_metadata.md) | Per-plugin global configuration (log formats, exporter endpoints, defaults). |

`apisix_ssl` is not yet implemented in this version; it will return in a future release.

## Highlights

- **Plugin Framework + plugin protocol v6** — the modern HashiCorp / OpenTofu provider stack.
- **Sparse `PUT` semantics** — removing a field from `.tf` removes it on the server. No silent merge / no leftover labels or plugins.
- **Drift-free JSON** — JSON-equivalence plan modifiers absorb APISIX's server-side normalization (key reordering, default injection) on plugin maps and JSON-string fields (`vars`, `health_check`).
- **Inline upstream parity** — `apisix_route.upstream` and `apisix_service.upstream` blocks expose the full upstream surface (scheme, timeouts, retries, hashing, keepalive, mTLS, service discovery, health checks), not just `type` and `nodes`.
- **Per-attribute validators** — HTTP methods, port ranges, status, type, scheme, hash_on, pass_host validated at plan time.
- **Per-resource Timeouts** — every resource accepts a `timeouts { create / read / update / delete }` block; default is 1 minute (vs Terraform's 20-minute framework default).
- **Resilient HTTP client** — exponential-backoff retries on transient 5xx and network errors for idempotent verbs; configurable timeout, optional TLS skip-verify, structured `tflog` request/response tracing under `TF_LOG=TRACE`.
- **Typed errors** — `client.ErrNotFound` sentinel for 404 detection (no string-matching).

## Requirements

- Terraform ≥ 1.0 or OpenTofu ≥ 1.6
- Apache APISIX ≥ 3.14
- Go ≥ 1.22 (only when building from source)

## Installation

### From the OpenTofu Registry

```hcl
terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "~> 0.3"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "your-admin-key"
}
```

The provider also reads `APISIX_BASE_URL` and `APISIX_ADMIN_KEY` from the environment if you prefer not to put them in HCL.

### From source

```bash
git clone https://github.com/scicore-unibas-ch/terraform-provider-apisix.git
cd terraform-provider-apisix
make build
```

## Provider Configuration

The `apisix` provider accepts the following arguments. Every argument is optional in the schema; `base_url` and `admin_key` are required in practice and may be supplied either in HCL or via environment variables.

| Argument | Environment variable | Description |
| --- | --- | --- |
| `base_url` | `APISIX_BASE_URL` | Base URL of the APISIX Admin API, e.g. `http://localhost:9180/apisix/admin`. |
| `admin_key` | `APISIX_ADMIN_KEY` | Admin API key (sensitive). |
| `timeout` | — | HTTP client timeout in seconds. Default: `30`. |
| `insecure` | — | Skip TLS certificate verification for HTTPS endpoints. Default: `false`. |

```hcl
provider "apisix" {
  base_url  = "https://apisix.internal:9180/apisix/admin"
  admin_key = var.apisix_admin_key
  timeout   = 30
  insecure  = false
}
```

Leaving `base_url` / `admin_key` out of the block falls back to `APISIX_BASE_URL` / `APISIX_ADMIN_KEY`, which keeps the admin key out of your configuration and state.

## Quick start

```hcl
resource "apisix_upstream" "backend" {
  id   = "backend-service"
  type = "roundrobin"

  nodes = [
    { host = "10.0.1.10", port = 8080, weight = 100 },
    { host = "10.0.1.11", port = 8080, weight = 50 },
  ]

  timeout = {
    connect = 5
    send    = 10
    read    = 30
  }
}

resource "apisix_route" "api" {
  id  = "api-route"
  uri = "/api/*"

  upstream_id = apisix_upstream.backend.id

  plugins = {
    "limit-count" = jsonencode({
      count         = 1000
      time_window   = 60
      rejected_code = 429
      key           = "remote_addr"
    })
  }

  timeouts {
    create = "30s"
    read   = "10s"
    update = "30s"
    delete = "10s"
  }
}
```

### Consumer authentication

```hcl
resource "apisix_consumer" "api_user" {
  id = "api-user-1"

  plugins = {
    "key-auth" = jsonencode({
      key = "secret-api-key-12345"
    })
  }
}

resource "apisix_route" "protected" {
  id  = "protected-route"
  uri = "/protected/*"

  upstream_id = apisix_upstream.backend.id

  plugins = {
    "key-auth" = jsonencode({})
  }
}
```

### Global rate limiting

```hcl
resource "apisix_global_rule" "rate_limit" {
  id = "global-rate-limit"

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

For more complete, runnable configurations — including advanced variants of every resource — see [`examples/resources/`](examples/resources/). Each resource has `apisix_<name>/basic/` and `apisix_<name>/advanced/` fixtures that apply cleanly against the docker-compose stack.

## Importing existing resources

Every resource supports `terraform import`, keyed by the APISIX Admin API `id`:

```bash
terraform import apisix_route.api api-route
terraform import apisix_upstream.backend backend-service
```

The `id` is the URL key under `/apisix/admin/<kind>/<id>` — the same value you set as `id` in HCL. See the **Import** section of each resource's page under [`docs/resources/`](docs/resources/) for per-resource details.

## Debugging

Set `TF_LOG=TRACE` (or `=DEBUG` for fewer details) to see structured request/response logs from the Admin-API client. TRACE includes the full request and response body — those bodies may contain TLS material or plugin secrets, so redirect TRACE output to a secure location in shared environments.

## Development

```bash
make build                                    # build the provider binary
make test                                     # go test ./...
make test-env-up                              # start the APISIX docker-compose cluster
make test-acceptance                          # run every acceptance test
make test-acceptance-single TEST=upstream     # run a single acceptance test
make test-env-down                            # stop the cluster
```

CI runs unit tests on every push and PR (`unit-tests.yml`) and full acceptance tests against four APISIX versions (3.14, 3.15, 3.16, 3.17) on `main` / PRs targeting `main` (`acceptance-tests.yml`). The acceptance workflow exercises both the legacy bash scripts under `tests/acceptance/` and Go acceptance tests (`TF_ACC=1`) that drive the provider through Terraform's own plan/apply lifecycle.

## Contributing

Pull requests welcome. Please:

1. Open an issue first for non-trivial changes.
2. Run `go vet ./...`, `go test ./...`, and `make test-acceptance` locally before submitting.
3. Match the conventional-commit style used in the existing history (`feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`, `ci:`, `release:`).

## License

MIT — see [LICENSE](LICENSE).

## Acknowledgments

- [Apache APISIX](https://apisix.apache.org/) — the API gateway this provider drives.
- [OpenTofu](https://opentofu.org/) and [Terraform](https://www.terraform.io/) — the IaC tools.
- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) — the provider SDK.
