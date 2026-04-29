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
        for id in route-test-upstream test-route-advanced test-route-basic test-route-complete test-route-with-script test-route-with-vars; do curl -s -X DELETE "http://localhost:9180/apisix/admin/routes/$id" -H "X-API-KEY: test123456789" > /dev/null 2>&1 || true; done
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
    for id in route-test-upstream test-route-advanced test-route-basic test-route-complete test-route-with-script test-route-with-vars; do curl -s -X DELETE "http://localhost:9180/apisix/admin/routes/$id" -H "X-API-KEY: test123456789" > /dev/null 2>&1 || true; done


# Test 1: Create all routes
log_info "Test 1: Create routes (basic, advanced, with_vars, complete, with_script)"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

# Verify all routes were created
for resource in basic advanced with_vars complete with_script; do
    ROUTE_ID=$(tofu state show apisix_route.$resource 2>&1 | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    if [ -z "$ROUTE_ID" ]; then
        log_error "Failed to get route ID for $resource"
        tofu state show apisix_route.$resource 2>&1 | head -5
        exit 1
    fi
    log_info "Route '$resource' created with ID: $ROUTE_ID"
    
    # Verify via APISIX API
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/routes/$ROUTE_ID" \
        -H "X-API-KEY: test123456789")
    if [ "$RESPONSE" != "200" ]; then
        log_error "Route '$resource' not found in APISIX (HTTP $RESPONSE)"
        curl -s "http://localhost:9180/apisix/admin/routes/$ROUTE_ID" -H "X-API-KEY: test123456789" | head -20
        exit 1
    fi
done
log_info "✓ All routes verified in APISIX API"

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

# Test 3: Verify route configuration via API
log_info "Test 3: Verify route configurations"
ADVANCED_ID=$(tofu state show apisix_route.advanced 2>/dev/null | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/routes/$ADVANCED_ID" -H "X-API-KEY: test123456789")

# Check for key fields using jq for proper JSON parsing
URIS=$(echo "$RESPONSE" | jq -r '.value.uris | length')
HOSTS=$(echo "$RESPONSE" | jq -r '.value.hosts | length')
METHODS=$(echo "$RESPONSE" | jq -r '.value.methods | length')
STATUS=$(echo "$RESPONSE" | jq -r '.value.status')

[ "$URIS" = "2" ] || { log_error "Advanced route URIs mismatch: got $URIS"; exit 1; }
[ "$HOSTS" = "2" ] || { log_error "Advanced route hosts mismatch: got $HOSTS"; exit 1; }
[ "$METHODS" = "2" ] || { log_error "Advanced route methods mismatch: got $METHODS"; exit 1; }
[ "$STATUS" = "1" ] || { log_error "Advanced route status mismatch: got $STATUS"; exit 1; }
log_info "✓ Route configurations verified"

# Test 3b: Verify complete route configuration
log_info "Test 3b: Verify complete route configuration"
COMPLETE_ID=$(tofu state show apisix_route.complete 2>/dev/null | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/routes/$COMPLETE_ID" -H "X-API-KEY: test123456789")

# Verify desc field
DESC=$(echo "$RESPONSE" | jq -r '.value.desc')
[ "$DESC" = "Complete route with all supported fields" ] || { log_error "Complete route desc mismatch: got $DESC"; exit 1; }

# Verify remote_addrs field
REMOTE_ADDRS=$(echo "$RESPONSE" | jq -r '.value.remote_addrs | length')
[ "$REMOTE_ADDRS" = "1" ] || { log_error "Complete route remote_addrs mismatch: got $REMOTE_ADDRS"; exit 1; }

# Verify enable_websocket field
WEBSOCKET=$(echo "$RESPONSE" | jq -r '.value.enable_websocket')
[ "$WEBSOCKET" = "true" ] || { log_error "Complete route websocket mismatch: got $WEBSOCKET"; exit 1; }

# Verify timeout fields
TIMEOUT_CONNECT=$(echo "$RESPONSE" | jq -r '.value.timeout.connect')
TIMEOUT_SEND=$(echo "$RESPONSE" | jq -r '.value.timeout.send')
TIMEOUT_READ=$(echo "$RESPONSE" | jq -r '.value.timeout.read')
[ "$TIMEOUT_CONNECT" = "5" ] || { log_error "Complete route timeout.connect mismatch: got $TIMEOUT_CONNECT"; exit 1; }
[ "$TIMEOUT_SEND" = "10" ] || { log_error "Complete route timeout.send mismatch: got $TIMEOUT_SEND"; exit 1; }
[ "$TIMEOUT_READ" = "15" ] || { log_error "Complete route timeout.read mismatch: got $TIMEOUT_READ"; exit 1; }

log_info "✓ Complete route configuration verified"

# Test 3c: Verify route with script
log_info "Test 3c: Verify route with script configuration"
WITH_SCRIPT_ID=$(tofu state show apisix_route.with_script 2>/dev/null | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/routes/$WITH_SCRIPT_ID" -H "X-API-KEY: test123456789")
SCRIPT=$(echo "$RESPONSE" | jq -r '.value.script')
if [ -z "$SCRIPT" ] || [ "$SCRIPT" = "null" ]; then
    log_error "with_script route script is empty or null"
    exit 1
fi
log_info "✓ Route with script configuration verified"

# Test 4: Destroy all routes
log_info "Test 4: Destroy routes"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

# Verify all routes were deleted
for resource in basic advanced with_vars complete with_script; do
    ROUTE_ID=$(tofu state show apisix_route.$resource 2>/dev/null | grep "^ *id *" | cut -d'"' -f2 || echo "")
    if [ -n "$ROUTE_ID" ]; then
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/routes/$ROUTE_ID" \
            -H "X-API-KEY: test123456789")
        if [ "$RESPONSE" != "404" ]; then
            log_error "Route '$resource' still exists in APISIX (HTTP $RESPONSE)"
            exit 1
        fi
    fi
done
log_info "✓ All routes deleted successfully"

# Test 5: Recreate all routes
log_info "Test 5: Recreate routes"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

for resource in basic advanced with_vars complete with_script; do
    ROUTE_ID=$(tofu state show apisix_route.$resource 2>/dev/null | grep "^ *id *" | cut -d'"' -f2)
    if [ -z "$ROUTE_ID" ]; then
        log_error "Failed to get route ID for $resource after recreation"
        exit 1
    fi
done
log_info "✓ All routes recreated successfully"

# Test 6: Import test for all routes
log_info "Test 6: Import test"
for resource in basic advanced with_vars complete with_script; do
    ROUTE_ID=$(tofu state show apisix_route.$resource 2>/dev/null | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    
    # Remove from state
    echo "Executing: tofu state rm apisix_route.$resource"
    tofu state rm apisix_route.$resource
    
    # Import back
    echo "Executing: tofu import apisix_route.$resource $ROUTE_ID"
    tofu import apisix_route.$resource "$ROUTE_ID"
    
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
