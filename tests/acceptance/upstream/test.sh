#!/bin/bash
set -e

# Configuration
CLEANUP_ON_FAILURE=${CLEANUP_ON_FAILURE:-false}
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$TEST_DIR"

# OpenTofu 1.11+ with dev_overrides doesn't need init
# Provider is configured via ~/.tofurc dev_overrides

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
# OpenTofu 1.11+ with dev_overrides: skip init, it's not necessary
# The provider is loaded from dev_overrides in ~/.tofurc
log_info "Using dev_overrides from ~/.tofurc (no init needed)"

# Test 1: Create all upstreams
log_info "Test 1: Create upstreams (basic, medium, complex)"
tofu apply -auto-approve -lock=false

# Verify all upstreams were created
for resource in basic medium complex; do
    UPSTREAM_ID=$(tofu state show apisix_upstream.$resource 2>/dev/null | grep "^ *id *" | cut -d'"' -f2)
    if [ -z "$UPSTREAM_ID" ]; then
        log_error "Failed to get upstream ID for $resource"
        exit 1
    fi
    log_info "Upstream '$resource' created with ID: $UPSTREAM_ID"
    
    # Verify via APISIX API
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/upstreams/$UPSTREAM_ID" \
        -H "X-API-KEY: test123456789")
    if [ "$RESPONSE" != "200" ]; then
        log_error "Upstream '$resource' not found in APISIX (HTTP $RESPONSE)"
        exit 1
    fi
done
log_info "✓ All upstreams verified in APISIX API"

# Test 2: Verify idempotency (should be no changes)
log_info "Test 2: Verify idempotency"
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

# Test 3: Verify specific fields in complex upstream
log_info "Test 3: Verify complex upstream configuration"
COMPLEX_ID=$(tofu state show apisix_upstream.complex 2>/dev/null | grep "^ *id *" | cut -d'"' -f2)
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/upstreams/$COMPLEX_ID" -H "X-API-KEY: test123456789")

# Check for key fields
echo "$RESPONSE" | grep -q '"type":"chash"' || { log_error "Complex upstream type mismatch"; exit 1; }
echo "$RESPONSE" | grep -q '"hash_on":"vars"' || { log_error "Complex upstream hash_on mismatch"; exit 1; }
echo "$RESPONSE" | grep -q '"key":"remote_addr"' || { log_error "Complex upstream key mismatch"; exit 1; }
echo "$RESPONSE" | grep -q '"retries":2' || { log_error "Complex upstream retries mismatch"; exit 1; }
log_info "✓ Complex upstream configuration verified"

# Test 4: Destroy all upstreams
log_info "Test 4: Destroy upstreams"
tofu destroy -auto-approve -lock=false

# Verify all upstreams were deleted
for resource in basic medium complex; do
    UPSTREAM_ID=$(tofu state show apisix_upstream.$resource 2>/dev/null | grep "^ *id *" | cut -d'"' -f2 || echo "")
    if [ -n "$UPSTREAM_ID" ]; then
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/upstreams/$UPSTREAM_ID" \
            -H "X-API-KEY: test123456789")
        if [ "$RESPONSE" != "404" ]; then
            log_error "Upstream '$resource' still exists in APISIX (HTTP $RESPONSE)"
            exit 1
        fi
    fi
done
log_info "✓ All upstreams deleted successfully"

# Test 5: Recreate all upstreams
log_info "Test 5: Recreate upstreams"
tofu apply -auto-approve -lock=false

for resource in basic medium complex; do
    UPSTREAM_ID=$(tofu state show apisix_upstream.$resource 2>/dev/null | grep "^ *id *" | cut -d'"' -f2)
    if [ -z "$UPSTREAM_ID" ]; then
        log_error "Failed to get upstream ID for $resource after recreation"
        exit 1
    fi
done
log_info "✓ All upstreams recreated successfully"

# Test 6: Import test for all upstreams
log_info "Test 6: Import test"
for resource in basic medium complex; do
    UPSTREAM_ID=$(tofu state show apisix_upstream.$resource 2>/dev/null | grep "^ *id *" | cut -d'"' -f2)
    
    # Remove from state
    tofu state rm apisix_upstream.$resource
    
    # Import back
    tofu import apisix_upstream.test "$UPSTREAM_ID"
    
    # Verify import worked (no changes after import)
    set +e
    tofu plan -detailed-exitcode -out=/dev/null -lock=false 2>&1 | grep -q "No changes"
    EXIT_CODE=$?
    set -e
    
    if [ $EXIT_CODE -ne 0 ]; then
        log_error "Import failed for $resource - state mismatch"
        tofu plan -lock=false
        exit 1
    fi
    log_info "✓ Import successful for $resource"
done

# Final cleanup
log_info "Cleanup"
tofu destroy -auto-approve -lock=false

log_info "✓ All acceptance tests passed!"
