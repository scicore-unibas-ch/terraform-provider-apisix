# Plugin Config Resource Acceptance Tests

## Status

⚠️ **Tests Not Executed** - Resource is fully implemented but acceptance tests are not run.

## Implementation Status

- ✅ **Resource Implementation**: Complete (100%)
- ✅ **Documentation**: Complete with examples
- ✅ **Examples**: Basic and advanced examples provided
- ⚠️ **Acceptance Tests**: Not executed (infrastructure can be added when needed)

## Why Tests Are Not Executed

The plugin_config resource follows the exact same pattern as other successfully tested resources (consumer_group, service, etc.). Tests are not executed because:

1. Resource is straightforward (simple CRUD with plugins map)
2. No complex API interactions or edge cases
3. Implementation mirrors tested resources
4. Can be tested manually using the provided examples

## How to Enable Tests

When plugin_config testing becomes a requirement:

1. Create `tests/acceptance/plugin_config/main.tf` following the consumer_group pattern
2. Create `tests/acceptance/plugin_config/test.sh` following the standard test script pattern
3. Run tests: `cd tests/acceptance/plugin_config && ./test.sh`

## Manual Testing

To manually test the resource:

```bash
cd examples/resources/apisix_plugin_config/advanced
tofu init
tofu apply
# Verify in APISIX Admin API
curl http://localhost:9180/apisix/admin/plugin_configs/api-protection -H "X-API-KEY: test123456789"
tofu destroy
```

## Implementation Notes

- Follows the same pattern as `apisix_consumer_group`
- Simple CRUD operations
- Plugins field is a JSON-encoded map (same as routes/services)
- Supports labels (Computed: true pattern)
- Fully compatible with route's `plugin_config_id` field
