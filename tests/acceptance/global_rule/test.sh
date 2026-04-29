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
    else
        log_warn "Leaving resources for debugging (set CLEANUP_ON_FAILURE=true to auto-cleanup)"
    fi
}

trap cleanup EXIT

# Initialize
log_info "Initializing Terraform..."
echo "Executing: tofu init -input=false"
tofu init -input=false

# Test 1: Create all global rules
log_info "Test 1: Create global rules (basic, multi_plugins, ip_restriction, route_integration)"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

# Verify all global rules were created
for resource in basic multi_plugins ip_restriction route_integration; do
    RULE_ID=$(tofu state show apisix_global_rule.$resource 2>&1 | grep '^\s*rule_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    if [ -z "$RULE_ID" ]; then
        log_error "Failed to get global rule ID for $resource"
        tofu state show apisix_global_rule.$resource 2>&1 | head -5
        exit 1
    fi
    log_info "Global rule '$resource' created with ID: $RULE_ID"
    
    # Verify via APISIX API
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/global_rules/$RULE_ID" \
        -H "X-API-KEY: test123456789")
    if [ "$RESPONSE" != "200" ]; then
        log_error "Global rule '$resource' not found in APISIX (HTTP $RESPONSE)"
        curl -s "http://localhost:9180/apisix/admin/global_rules/$RULE_ID" -H "X-API-KEY: test123456789" | head -20
        exit 1
    fi
done
log_info "✓ All global rules verified in APISIX API"

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

# Test 3: Verify global rule configurations via API
log_info "Test 3: Verify global rule configurations"

# Verify multi_plugins global rule
MULTI_ID=$(tofu state show apisix_global_rule.multi_plugins 2>/dev/null | grep '^\s*rule_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/global_rules/$MULTI_ID" -H "X-API-KEY: test123456789")
PLUGINS_COUNT=$(echo "$RESPONSE" | jq -r '.value.plugins | keys | length')
[ "$PLUGINS_COUNT" = "2" ] || { log_error "multi_plugins global rule plugins mismatch: got $PLUGINS_COUNT"; exit 1; }

# Verify ip_restriction global rule
IP_ID=$(tofu state show apisix_global_rule.ip_restriction 2>/dev/null | grep '^\s*rule_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/global_rules/$IP_ID" -H "X-API-KEY: test123456789")
BLACKLIST=$(echo "$RESPONSE" | jq -r '.value.plugins["ip-restriction"].blacklist | length')
[ "$BLACKLIST" = "1" ] || { log_error "ip_restriction global rule blacklist mismatch: got $BLACKLIST"; exit 1; }

log_info "✓ Global rule configurations verified"

# Test 4: Destroy all global rules
log_info "Test 4: Destroy global rules"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

# Verify all global rules were deleted
for resource in basic multi_plugins ip_restriction route_integration; do
    RULE_ID=$(tofu state show apisix_global_rule.$resource 2>/dev/null | grep "^ *rule_id *" | cut -d'"' -f2 || echo "")
    if [ -n "$RULE_ID" ]; then
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/global_rules/$RULE_ID" \
            -H "X-API-KEY: test123456789")
        if [ "$RESPONSE" != "404" ]; then
            log_error "Global rule '$resource' still exists in APISIX (HTTP $RESPONSE)"
            exit 1
        fi
    fi
done
log_info "✓ All global rules deleted successfully"

# Test 5: Recreate all global rules
log_info "Test 5: Recreate global rules"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

for resource in basic multi_plugins ip_restriction route_integration; do
    RULE_ID=$(tofu state show apisix_global_rule.$resource 2>/dev/null | grep "^ *rule_id *" | cut -d'"' -f2)
    if [ -z "$RULE_ID" ]; then
        log_error "Failed to get global rule ID for $resource after recreation"
        exit 1
    fi
done
log_info "✓ All global rules recreated successfully"

# Test 6: Import test for all global rules
log_info "Test 6: Import test"
for resource in basic multi_plugins ip_restriction route_integration; do
    RULE_ID=$(tofu state show apisix_global_rule.$resource 2>/dev/null | grep '^\s*rule_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    
    # Remove from state
    echo "Executing: tofu state rm apisix_global_rule.$resource"
    tofu state rm apisix_global_rule.$resource
    
    # Import back
    echo "Executing: tofu import apisix_global_rule.$resource $RULE_ID"
    tofu import apisix_global_rule.$resource "$RULE_ID"
    
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
