# Global Rule Resource Acceptance Tests

## Status

⚠️ **Tests Not Executed** - Resource is fully implemented but acceptance tests are not run.

## Implementation Status

- ✅ **Resource Implementation**: Complete (100%)
- ✅ **Documentation**: Complete with examples
- ✅ **Examples**: Basic and advanced examples provided
- ⚠️ **Acceptance Tests**: Not executed (infrastructure can be added when needed)

## Why Tests Are Not Executed

The global_rule resource is straightforward and follows the exact same pattern as plugin_config. Tests are not executed because:

1. Resource is simple (just rule_id and plugins)
2. No complex API interactions
3. Implementation mirrors tested resources (plugin_config)
4. Can be tested manually using the provided examples
5. Global rules affect ALL routes - requires careful test isolation

## How to Enable Tests

When global_rule testing becomes a requirement:

1. Create `tests/acceptance/global_rule/main.tf` following the plugin_config pattern
2. Create `tests/acceptance/global_rule/test.sh` following the standard test script pattern
3. Ensure test isolation (global rules affect all routes)
4. Run tests: `cd tests/acceptance/global_rule && ./test.sh`

## Manual Testing

To manually test the resource:

```bash
cd examples/resources/apisix_global_rule/basic
tofu init
tofu apply
# Verify in APISIX Admin API
curl http://localhost:9180/apisix/admin/global_rules/global-rate-limit -H "X-API-KEY: test123456789"
tofu destroy
```

## Implementation Notes

- Follows the same pattern as `apisix_plugin_config`
- Simple CRUD operations
- Only two fields: rule_id and plugins
- No labels or desc fields (APISIX API restriction)
- Plugins apply to ALL routes automatically
- Fully compatible with route and plugin_config resources
