#!/bin/bash
set -e

# Configuration
CLEANUP_ON_FAILURE=${CLEANUP_ON_FAILURE:-true}
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$TEST_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}=== $1${NC}"
}

log_error() {
    echo -e "${RED}✗ $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

cleanup() {
    if [ "$CLEANUP_ON_FAILURE" = "true" ] || [ $? -eq 0 ]; then
        log_info "Cleaning up..."
        tofu destroy -auto-approve -lock=false 2>/dev/null || true
        for id in test-pc-basic test-pc-labels test-pc-multi test-pc-route; do curl -s -X DELETE "http://localhost:9180/apisix/admin/plugin_configs/$id" -H "X-API-KEY: test123456789" > /dev/null 2>&1 || true; done
    else
        log_warn "Leaving resources for debugging (set CLEANUP_ON_FAILURE=true to auto-cleanup)"
    fi
}

trap cleanup EXIT

# Initialize
log_info "Initializing Terraform..."
# echo "Executing: tofu init -input=false"
# tofu init -input=false

# Initial cleanup
log_info "Cleaning up any existing state and APISIX resources..."
tofu destroy -auto-approve -lock=false 2>/dev/null || true
    for id in test-pc-basic test-pc-labels test-pc-multi test-pc-route; do curl -s -X DELETE "http://localhost:9180/apisix/admin/plugin_configs/$id" -H "X-API-KEY: test123456789" > /dev/null 2>&1 || true; done


# Test 1: Create all plugin configs
log_info "Test 1: Create plugin configs (basic, multi_plugins, with_labels, route_integration)"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

# Verify all plugin configs were created
for resource in basic multi_plugins with_labels route_integration; do
    CONFIG_ID=$(tofu state show apisix_plugin_config.$resource 2>&1 | grep '^\s*config_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    if [ -z "$CONFIG_ID" ]; then
        log_error "Failed to get plugin config ID for $resource"
        tofu state show apisix_plugin_config.$resource 2>&1 | head -5
        exit 1
    fi
    log_info "Plugin config '$resource' created with ID: $CONFIG_ID"
    
    # Verify via APISIX API
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/plugin_configs/$CONFIG_ID" \
        -H "X-API-KEY: test123456789")
    if [ "$RESPONSE" != "200" ]; then
        log_error "Plugin config '$resource' not found in APISIX (HTTP $RESPONSE)"
        curl -s "http://localhost:9180/apisix/admin/plugin_configs/$CONFIG_ID" -H "X-API-KEY: test123456789" | head -20
        exit 1
    fi
done
log_info "✓ All plugin configs verified in APISIX API"

# Test 2: Verify idempotency (should be no changes)
log_info "Test 2: Verify idempotency"
echo "Executing: tofu plan -detailed-exitcode -lock=false"
set +e
PLAN_OUTPUT=$(tofu plan -detailed-exitcode -lock=false 2>&1)
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 0 ]; then
    log_info "✓ No changes detected (idempotent)"
else
    log_error "Changes detected - NOT idempotent!"
    echo "$PLAN_OUTPUT"
    exit 1
fi

# Test 3: Verify plugin config configurations via API
log_info "Test 3: Verify plugin config configurations"

# Verify multi_plugins plugin config
MULTI_ID=$(tofu state show apisix_plugin_config.multi_plugins 2>/dev/null | grep '^\s*config_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/plugin_configs/$MULTI_ID" -H "X-API-KEY: test123456789")
PLUGINS_COUNT=$(echo "$RESPONSE" | jq -r '.value.plugins | keys | length')
[ "$PLUGINS_COUNT" = "2" ] || { log_error "multi_plugins plugin config plugins mismatch: got $PLUGINS_COUNT"; exit 1; }

# Verify with_labels plugin config
WITH_LABELS_ID=$(tofu state show apisix_plugin_config.with_labels 2>/dev/null | grep '^\s*config_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/plugin_configs/$WITH_LABELS_ID" -H "X-API-KEY: test123456789")
LABELS_COUNT=$(echo "$RESPONSE" | jq -r '.value.labels | keys | length')
[ "$LABELS_COUNT" = "3" ] || { log_error "with_labels plugin config labels mismatch: got $LABELS_COUNT"; exit 1; }

# Verify route integration
ROUTE_ID=$(tofu state show apisix_route.with_plugin_config 2>/dev/null | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/routes/$ROUTE_ID" -H "X-API-KEY: test123456789")
PLUGIN_CONFIG_ID=$(echo "$RESPONSE" | jq -r '.value.plugin_config_id')
[ "$PLUGIN_CONFIG_ID" = "test-pc-route" ] || { log_error "Route plugin_config_id mismatch: got $PLUGIN_CONFIG_ID"; exit 1; }

log_info "✓ Plugin config configurations verified"

# Test 4: Destroy all plugin configs
log_info "Test 4: Destroy plugin configs"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

# Verify all plugin configs were deleted
for resource in basic multi_plugins with_labels route_integration; do
    CONFIG_ID=$(tofu state show apisix_plugin_config.$resource 2>/dev/null | grep "^ *config_id *" | cut -d'"' -f2 || echo "")
    if [ -n "$CONFIG_ID" ]; then
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/plugin_configs/$CONFIG_ID" \
            -H "X-API-KEY: test123456789")
        if [ "$RESPONSE" != "404" ]; then
            log_error "Plugin config '$resource' still exists in APISIX (HTTP $RESPONSE)"
            exit 1
        fi
    fi
done
log_info "✓ All plugin configs deleted successfully"

# Test 5: Recreate all plugin configs
log_info "Test 5: Recreate plugin configs"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

for resource in basic multi_plugins with_labels route_integration; do
    CONFIG_ID=$(tofu state show apisix_plugin_config.$resource 2>/dev/null | grep "^ *config_id *" | cut -d'"' -f2)
    if [ -z "$CONFIG_ID" ]; then
        log_error "Failed to get plugin config ID for $resource after recreation"
        exit 1
    fi
done
log_info "✓ All plugin configs recreated successfully"

# Test 6: Import test for all plugin configs
log_info "Test 6: Import test"
for resource in basic multi_plugins with_labels route_integration; do
    CONFIG_ID=$(tofu state show apisix_plugin_config.$resource 2>/dev/null | grep '^\s*config_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    
    # Remove from state
    echo "Executing: tofu state rm apisix_plugin_config.$resource"
    tofu state rm apisix_plugin_config.$resource
    
    # Import back
    echo "Executing: tofu import apisix_plugin_config.$resource $CONFIG_ID"
    tofu import apisix_plugin_config.$resource "$CONFIG_ID"
    
    # Verify import worked (no changes after import)
    echo "Executing: tofu plan -detailed-exitcode -out=/dev/null -lock=false"
    set +e
    tofu plan -detailed-exitcode -out=/dev/null -lock=false 2>&1 | grep -q "No changes"
    EXIT_CODE=$?
    set -e
    
    if [ $EXIT_CODE -ne 0 ]; then
        log_error "Import failed for $resource - state mismatch"
        tofu plan -lock=false | head -30
        exit 1
    fi
    
    # Additional verification: run tofu apply to ensure no changes
    echo "Executing: tofu apply -auto-approve -lock=false (verify no changes after import)"
    APPLY_OUTPUT=$(tofu apply -auto-approve -lock=false 2>&1)
    if echo "$APPLY_OUTPUT" | grep -q "Resources: 0 added, 0 changed, 0 destroyed"; then
        log_info "✓ Import successful for $resource (verified with apply)"
    else
        log_error "Import verification failed for $resource - apply detected changes"
        echo "$APPLY_OUTPUT"
        exit 1
    fi
    
    # Verify idempotency after import (plan should show no changes)
    echo "Executing: tofu plan -detailed-exitcode -lock=false (verify idempotency after import)"
    set +e
    PLAN_OUTPUT=$(tofu plan -detailed-exitcode -lock=false 2>&1)
    EXIT_CODE=$?
    set -e
    
    if [ $EXIT_CODE -ne 0 ]; then
        log_error "Import idempotency check failed for $resource - plan detected changes"
        echo "$PLAN_OUTPUT"
        exit 1
    fi
    log_info "✓ Import idempotency verified for $resource"
done

# Final cleanup
log_info "Cleanup"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

log_info "✓ All acceptance tests passed!"
