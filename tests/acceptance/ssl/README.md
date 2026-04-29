# SSL Resource Acceptance Tests

## Requirements

The `apisix_ssl` resource requires APISIX to be configured with SSL proxy enabled. The default test environment does not include SSL proxy configuration.

## Enabling SSL Tests

To run SSL acceptance tests, you need to:

1. **Enable SSL proxy in APISIX configuration** (`tests/apisix/config.yaml`):
   ```yaml
   apisix:
     ssl:
       enable: true
       listen:
         - port: 9443
           enable_http2: true
   ```

2. **Provide test certificates**:
   - Create self-signed certificates for testing
   - Place them in `tests/acceptance/ssl/` directory
   - Or use environment variables

3. **Update test configuration** to use the SSL-enabled APISIX port

## Test Structure

Once SSL is enabled, the test structure should follow the pattern:

- `main.tf` - Test SSL configurations (basic, multi-sni, mtls, etc.)
- `test.sh` - Acceptance test script following the standard pattern:
  - Create
  - Verify idempotency
  - Verify via API
  - Destroy
  - Recreate
  - Import test

## Current Status

⚠️ **Tests not implemented** - SSL proxy not enabled in test environment

The resource implementation is complete and follows the same pattern as other resources. Tests can be added when SSL proxy is enabled in the test environment.

## Manual Testing

To manually test the SSL resource:

1. Enable SSL proxy in your APISIX deployment
2. Create test certificates
3. Run:
   ```bash
   cd tests/acceptance/ssl
   tofu init
   tofu apply
   curl -k https://localhost:9443 -H "Host: example.com"
   ```

## Implementation Notes

- The `cert` and `key` fields are marked as `Sensitive: true`
- The API returns masked certificate data, so we don't read them back in `flattenSSL`
- The `client` block enables mTLS (mutual TLS) configuration
- Multiple SNIs are supported via the `snis` list field
- SSL protocols can be restricted (e.g., TLSv1.3 only)
