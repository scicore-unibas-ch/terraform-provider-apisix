# APISIX Upstream Examples

This directory contains example Terraform configurations for the `apisix_upstream` resource.

## Examples

### Basic Upstream

The `basic/` example demonstrates a simple upstream with minimal configuration:

- Single node
- Round-robin load balancing
- Basic labels

**Use case:** Simple backend services with a single instance.

```bash
cd basic
tofu init
tofu apply
```

### Advanced Upstream

The `advanced/` example demonstrates a complete upstream configuration:

- Multiple nodes with different weights and priorities
- Active and passive health checks
- Timeout configuration
- Keepalive pool
- Service discovery (Consul)
- mTLS support
- Complete labeling

**Use case:** Production-ready upstream configurations with high availability and advanced features.

```bash
cd advanced
tofu init
tofu apply
```

## Running the Examples

1. Ensure you have OpenTofu installed
2. Ensure APISIX is running and accessible
3. Set your environment variables or update the provider configuration:
   ```bash
   export APISIX_BASE_URL="http://localhost:9180/apisix/admin"
   export APISIX_ADMIN_KEY="your-admin-key"
   ```
4. Run `tofu init` and `tofu apply` in the example directory

## Cleaning Up

Remember to destroy the resources when you're done:

```bash
tofu destroy
```
