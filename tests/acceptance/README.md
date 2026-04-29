# Manual Testing Guide

## Why Manual Testing?

OpenTofu 1.11+ requires providers to be either:
1. Published to the OpenTofu Registry, OR
2. Available in a local mirror with proper version metadata

Our provider (`scicore/apisix`) is not yet published to the registry, so automated acceptance tests cannot run locally with `tofu apply`. 

**Note:** The `pescobar/slurm` provider works because it's published to the registry (v0.1.3), even though the binary is overridden via `dev_overrides`.

## Prerequisites

1. Docker test environment running:
   ```bash
   make test-env-up
   ```

2. Provider binary built:
   ```bash
   make build
   ```

## Manual Test 1: Create Upstream via API

```bash
# Create a basic upstream
curl -X PUT http://localhost:9180/apisix/admin/upstreams/manual-test-1 \
  -H "X-API-KEY: test123456789" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "manual-test-1",
    "type": "roundrobin",
    "desc": "Manual test upstream",
    "nodes": [
      {"host": "127.0.0.1", "port": 8080, "weight": 100}
    ],
    "scheme": "http",
    "labels": {
      "env": "test",
      "team": "platform"
    }
  }'

# Verify creation
curl http://localhost:9180/apisix/admin/upstreams/manual-test-1 \
  -H "X-API-KEY: test123456789" | jq .

# Expected response should show the upstream configuration
```

## Manual Test 2: Create Complex Upstream

```bash
# Create upstream with all fields
curl -X PUT http://localhost:9180/apisix/admin/upstreams/manual-test-complex \
  -H "X-API-KEY: test123456789" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "manual-test-complex",
    "type": "chash",
    "hash_on": "vars",
    "key": "remote_addr",
    "nodes": [
      {"host": "127.0.0.1", "port": 9080, "weight": 100, "priority": 0},
      {"host": "127.0.0.1", "port": 9081, "weight": 50, "priority": 1}
    ],
    "timeout": {
      "connect": 3,
      "send": 5,
      "read": 10
    },
    "retries": 2,
    "retry_timeout": 5,
    "scheme": "http",
    "pass_host": "pass",
    "labels": {
      "env": "test",
      "complexity": "high"
    }
  }'

# Verify
curl http://localhost:9180/apisix/admin/upstreams/manual-test-complex \
  -H "X-API-KEY: test123456789" | jq '.value'
```

## Manual Test 3: Update Upstream

```bash
# Update the upstream
curl -X PATCH http://localhost:9180/apisix/admin/upstreams/manual-test-1 \
  -H "X-API-KEY: test123456789" \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": [
      {"host": "127.0.0.1", "port": 8080, "weight": 100},
      {"host": "127.0.0.1", "port": 8081, "weight": 50}
    ]
  }'

# Verify update
curl http://localhost:9180/apisix/admin/upstreams/manual-test-1 \
  -H "X-API-KEY: test123456789" | jq '.value.nodes'
```

## Manual Test 4: Delete Upstream

```bash
# Delete the upstream
curl -X DELETE http://localhost:9180/apisix/admin/upstreams/manual-test-1 \
  -H "X-API-KEY: test123456789"

# Verify deletion (should return 404)
curl -i http://localhost:9180/apisix/admin/upstreams/manual-test-1 \
  -H "X-API-KEY: test123456789" | grep "HTTP/"
```

## Testing with Terraform/OpenTofu (Future)

Once the provider is published to a registry, run automated tests:

```bash
# Run all acceptance tests
make test-acceptance

# Run upstream tests only
make test-acceptance-single TEST=upstream

# Run with cleanup on failure
CLEANUP_ON_FAILURE=true make test-acceptance-single TEST=upstream
```

## Cleanup

```bash
# Stop Docker environment
make test-env-down
```

## Troubleshooting

### OpenTofu dev_overrides not working

OpenTofu 1.11+ has stricter handling of `dev_overrides`. Solutions:

1. **Use older OpenTofu** (< 1.10) for local development
2. **Publish provider** to a local/remote registry
3. **Test via API** using curl commands above

### Provider not found

Ensure `~/.tofurc` contains:

```hcl
provider_installation {
  dev_overrides {
    "scicore/apisix" = "/home/escobar/github/terraform-provider-apisix"
  }
  direct {}
}
```

And rebuild the provider:

```bash
make build
```
