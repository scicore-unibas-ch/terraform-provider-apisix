# APISIX Terraform Provider - Implementation Status

## Overview

This document tracks the implementation status of APISIX resources and fields in the Terraform provider. Use this as a reference for future development priorities.

## Implemented Resources (Complete)

### ✅ apisix_upstream
**Status:** 97% complete  
**File:** `internal/resources/resource_apisix_upstream.go`  
**Tests:** `tests/acceptance/upstream/`  
**Documentation:** `docs/resources/upstream.md`

**All fields implemented except:**
- `tls.client_cert_id` - Requires pre-existing SSL certificate resource
- `upstream_host` - Specialized field for host rewriting (pass_host=rewrite mode)

### ✅ apisix_route
**Status:** 93% complete  
**File:** `internal/resources/resource_apisix_route.go`  
**Tests:** `tests/acceptance/route/`  
**Documentation:** `docs/resources/route.md`

**All fields implemented except:**
- `filter_func` - Requires custom Lua function code
- `service_id` - Requires pre-existing Service resource (dependency)
- `plugin_config_id` - Requires pre-existing Plugin Config resource (dependency)

### ✅ apisix_service
**Status:** 100% complete  
**File:** `internal/resources/resource_apisix_service.go`  
**Tests:** `tests/acceptance/service/`  
**Documentation:** `docs/resources/service.md`

**All fields implemented and tested.**

### ✅ apisix_consumer
**Status:** 100% complete  
**File:** `internal/resources/resource_apisix_consumer.go`  
**Tests:** `tests/acceptance/consumer/`  
**Documentation:** `docs/resources/consumer.md`

**All fields implemented and tested.**

### ✅ apisix_consumer_group
**Status:** 100% complete  
**File:** `internal/resources/resource_apisix_consumer_group.go`  
**Tests:** `tests/acceptance/consumer_group/`  
**Documentation:** `docs/resources/consumer_group.md`

**All fields implemented and tested.**

---

## ✅ apisix_ssl
**Status:** IMPLEMENTED (tests pending SSL proxy enablement)  
**File:** `internal/resources/resource_apisix_ssl.go`  
**Tests:** `tests/acceptance/ssl/README.md` (documentation only - requires SSL proxy)  
**Documentation:** `docs/resources/ssl.md`

**All fields implemented:**
- ✅ `sni` - Primary SNI
- ✅ `snis` - List of SNIs
- ✅ `cert` - SSL certificate (sensitive)
- ✅ `key` - SSL private key (sensitive)
- ✅ `certs` - Multiple certificates for SNI
- ✅ `keys` - Multiple keys for SNI
- ✅ `ssl_protocols` - TLS version configuration
- ✅ `client` - mTLS client verification (ca_cert, depth)
- ✅ `labels` - Resource labels

**Implementation Notes:**
- Certificate and key are marked as sensitive
- API returns masked certificate data (not read back)
- Full support for mTLS via `client` block
- Tests require SSL proxy enabled in APISIX (not enabled in default test environment)
- See `tests/acceptance/ssl/README.md` for enabling SSL tests

---

### ⏳ apisix_plugin_config
**Priority:** MEDIUM  
**API Docs:** https://apisix.apache.org/docs/apisix/admin-api/#plugin-config

**Purpose:** Reusable plugin configurations that can be referenced by routes

**Expected Fields:**
- `id` - Plugin config ID
- `desc` - Description
- `plugins` - Plugin configurations (same format as routes)
- `labels` - Resource labels

**Implementation Notes:**
- Similar to Service but for plugin configurations only
- Routes reference via `plugin_config_id` field
- Useful for applying same plugin config to multiple routes

---

### ⏳ apisix_global_rule
**Priority:** MEDIUM  
**API Docs:** https://apisix.apache.org/docs/apisix/admin-api/#global-rule

**Purpose:** Global plugins/rules applied to all requests

**Expected Fields:**
- `id` - Global rule ID (usually "global_rules")
- `plugins` - Plugin configurations

**Implementation Notes:**
- Only one global rule resource typically exists
- Plugins apply to ALL routes automatically
- Useful for global rate limiting, logging, etc.

---

### ⏳ apisix_stream_route
**Priority:** LOW  
**API Docs:** https://apisix.apache.org/docs/apisix/admin-api/#stream-route

**Purpose:** Layer 4 (TCP/UDP) routing

**Expected Fields:**
- `id` - Stream route ID
- `sni` - Server Name Indication
- `server_port` - Listening port
- `upstream_id` - Reference to upstream
- `upstream` - Inline upstream configuration
- `plugins` - Plugin configurations
- `labels` - Resource labels

**Implementation Notes:**
- Different from regular routes (Layer 4 vs Layer 7)
- Requires APISIX stream proxy enabled
- Useful for TCP/UDP traffic routing

---

### ⏳ apisix_system_config
**Priority:** LOW  
**API Docs:** https://apisix.apache.org/docs/apisix/admin-api/#system-config

**Purpose:** APISIX system-wide configuration

**Expected Fields:**
- Various system configuration options

**Implementation Notes:**
- Advanced use case
- May conflict with APISIX deployment configuration
- Use with caution

---

### ⏳ apisix_route_match_expr
**Priority:** LOW  
**API Docs:** https://apisix.apache.org/docs/apisix/admin-api/#route

**Purpose:** Advanced route matching with expressions

**Expected Fields:**
- Expression-based route matching (alternative to vars)

**Implementation Notes:**
- More powerful than vars filtering
- Requires understanding of APISIX expression syntax
- Advanced feature

---

## Field Implementation Gaps

### Routes
- `filter_func` - Custom Lua function for filtering (requires Lua code)
- `service_id` - Requires Service resource implementation (DONE)
- `plugin_config_id` - Requires Plugin Config resource (NOT YET IMPLEMENTED)

### Upstreams
- `tls.client_cert_id` - Requires SSL resource implementation (NOT YET IMPLEMENTED)
- `upstream_host` - Only needed when pass_host="rewrite" (specialized use case)

---

## Implementation Priority Recommendations

### Phase 1 (High Priority)
1. ✅ **apisix_ssl** - IMPLEMENTED (tests pending SSL proxy enablement)
2. **apisix_plugin_config** - Useful for DRY configurations

### Phase 2 (Medium Priority)
3. **apisix_global_rule** - Useful for global policies
4. **Add remaining route dependencies** - service_id, plugin_config_id references

### Phase 3 (Low Priority / On Demand)
5. **apisix_stream_route** - Only if Layer 4 routing needed
6. **apisix_system_config** - Only if system config management needed
7. **Advanced fields** - filter_func, upstream_host, etc.

---

## Testing Requirements for New Resources

When implementing new resources, follow the established pattern:

1. **Resource Implementation** (`internal/resources/resource_apisix_<name>.go`)
   - Full CRUD operations
   - Proper error handling
   - Labels support (Computed: true, always set)
   - Import support

2. **Acceptance Tests** (`tests/acceptance/<name>/`)
   - `main.tf` - Multiple resource configurations testing different fields
   - `test.sh` - Full test suite (create, idempotency, API verification, destroy, recreate, import)
   - All tests must pass

3. **Documentation** (`docs/resources/<name>.md`)
   - Example usage (basic and advanced)
   - Complete argument reference
   - Import instructions

4. **Examples** (`examples/resources/apisix_<name>/`)
   - `basic/` - Simple example
   - `advanced/` - All fields demonstrated

5. **Field Coverage Verification**
   - All schema fields should have test coverage
   - All schema fields should appear in at least one example
   - All schema fields documented

---

## Development Notes

### Provider Pattern
- Use `utils.go` for shared helper functions
- Follow existing resource naming conventions
- Use `ForceNew: true` for immutable fields (like username, group_id)
- Always set labels even when empty (Computed: true pattern)

### Testing
- Use Docker Compose environment (APISIX 3.16.0 + etcd 3.5.9)
- Test via APISIX Admin API (port 9180)
- Admin key: `test123456789`
- Clean up resources after tests

### Building
```bash
make build
cp terraform-provider-apisix ~/.tofu.d/plugins/registry.opentofu.org/scicore/apisix/0.1.0/linux_amd64/
```

### Running Tests
```bash
cd tests/acceptance/<resource>
./test.sh
```

---

## Last Updated

2026-04-29

**Current Status:** 6 resources implemented (5 complete, 1 with tests pending)  
**Total APISIX Resources:** 12+  
**Coverage:** ~50% of all APISIX resources

### Recent Additions
- **apisix_ssl** - SSL/TLS certificate management (implementation complete, tests require SSL proxy)
