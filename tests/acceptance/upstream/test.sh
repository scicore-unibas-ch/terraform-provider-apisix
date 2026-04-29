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
        # Remove temporary config
        rm -f .tofurc terraform.tfstate* terraform.tfstate.backup 2>/dev/null || true
        rm -rf .terraform* 2>/dev/null || true
    else

        # Force cleanup via API (in case state is corrupted)
        log_info "Force cleaning upstream via API..."
        for rid in test-upstream-basic test-upstream-medium test-upstream-complex; do
            curl -s -X DELETE "http://localhost:9180/apisix/admin/upstreams/$rid" \
                -H "X-API-KEY: test123456789" > /dev/null 2>&1 || true
        done
        log_warn "Leaving resources for debugging (set CLEANUP_ON_FAILURE=true to auto-cleanup)"
    fi
}

trap cleanup EXIT

# Generate temporary .tofurc for this test
log_info "Generating temporary provider config..."
PROVIDER_DIR="$(cd "$TEST_DIR/../../.." && pwd)"
cat > "$TEST_DIR/.tofurc" << TOFURC
provider_installation {
  dev_overrides {
    "scicore-unibas-ch/apisix" = "$PROVIDER_DIR"
  }
  direct {}
}
TOFURC
export TF_CLI_CONFIG_FILE="$TEST_DIR/.tofurc"
log_info "Using config: $TF_CLI_CONFIG_FILE"



# Initialize
log_info "Initializing Terraform..."

# Restart APISIX for clean state
log_info "Restarting APISIX cluster for clean state..."
cd ../../
docker compose down --volumes --remove-orphans >/dev/null 2>&1 || true
docker compose up -d >/dev/null 2>&1
sleep 8
cd - >/dev/null

# Wait for APISIX to be ready
log_info "Waiting for APISIX to be ready..."
for i in {1..60}; do
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/routes" \
        -H "X-API-KEY: test123456789" | grep -q "200"; then
        log_info "APISIX is ready"
        break
    fi
    sleep 1
done

# Remove lock files for clean test
rm -f .terraform.lock.hcl 2>/dev/null || true

# Test 1: Create upstreams (basic, medium, complex)
log_info "Test 1: Create upstreams (basic, medium, complex)"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

# Verify all upstreams were created
for resource in basic medium complex; do
    UPSTREAM_ID=$(tofu state show apisix_upstream.$resource 2>&1 | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    if [ -z "$UPSTREAM_ID" ]; then
        log_error "Failed to get upstream ID for $resource"
        tofu state show apisix_upstream.$resource 2>&1 | head -5
        exit 1
    fi
    log_info "Upstream '$resource' created with ID: $UPSTREAM_ID"
    
    # Verify via APISIX API
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/upstreams/$UPSTREAM_ID" \
        -H "X-API-KEY: test123456789")
    if [ "$RESPONSE" != "200" ]; then
        log_error "Upstream '$resource' not found in APISIX (HTTP $RESPONSE)"
        curl -s "http://localhost:9180/apisix/admin/upstreams/$UPSTREAM_ID" -H "X-API-KEY: test123456789" | head -20
        exit 1
    fi
done
log_info "✓ All upstreams verified in APISIX API"

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

# Test 3: Verify upstream configurations via API
log_info "Test 3: Verify upstream configurations"

# Verify medium upstream (multiple nodes)
MEDIUM_ID=$(tofu state show apisix_upstream.medium 2>/dev/null | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/upstreams/$MEDIUM_ID" -H "X-API-KEY: test123456789")
NODES_COUNT=$(echo "$RESPONSE" | jq -r '.value.nodes | length')
[ "$NODES_COUNT" = "2" ] || { log_error "medium upstream nodes mismatch: got $NODES_COUNT"; exit 1; }

# Verify complex upstream (has labels)
COMPLEX_ID=$(tofu state show apisix_upstream.complex 2>/dev/null | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/upstreams/$COMPLEX_ID" -H "X-API-KEY: test123456789")
LABELS=$(echo "$RESPONSE" | jq -r '.value.labels | length')
[ "$LABELS" -ge "1" ] || { log_error "complex upstream labels mismatch: got $LABELS"; exit 1; }

log_info "✓ Upstream configurations verified"

# Test 4: Destroy all upstreams
log_info "Test 4: Destroy upstreams"
echo "Executing: tofu destroy -auto-approve -lock=false"
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
echo "Executing: tofu apply -auto-approve -lock=false"
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
    UPSTREAM_ID=$(tofu state show apisix_upstream.$resource 2>/dev/null | grep '^\s*id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    
    # Remove from state
    echo "Executing: tofu state rm apisix_upstream.$resource"
    tofu state rm apisix_upstream.$resource
    
    # Import back
    echo "Executing: tofu import apisix_upstream.$resource $UPSTREAM_ID"
    tofu import apisix_upstream.$resource "$UPSTREAM_ID"
    
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
log_info "Final cleanup"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

log_info "✓ All acceptance tests passed!"
