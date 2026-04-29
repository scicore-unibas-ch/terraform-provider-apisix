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
# echo "Executing: tofu init -input=false"
# tofu init -input=false

# Remove lock files for clean test
rm -f .terraform.lock.hcl 2>/dev/null || true

# Remove lock files for clean test
rm -f .terraform.lock.hcl 2>/dev/null || true

# Restart APISIX for clean state
log_info "Restarting APISIX cluster for clean state..."
cd ../../
docker compose down -v >/dev/null 2>&1 || true
docker compose up -d >/dev/null 2>&1
sleep 8
cd - >/dev/null

# Wait for APISIX to be ready
for i in {1..60}; do
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/routes" \
        -H "X-API-KEY: test123456789" | grep -q "200"; then
        log_info "APISIX is ready"
        break
    fi
    sleep 1
done

# Initial cleanup
log_info "Cleaning up any existing state and APISIX resources..."
tofu destroy -auto-approve -lock=false 2>/dev/null || true


# Test 1: Create all consumers
log_info "Test 1: Create consumers (basic, key_auth, jwt_auth, with_labels, hmac_auth, with_group)"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

# Verify all consumers were created
for resource in basic key_auth jwt_auth with_labels hmac_auth with_group; do
    CONSUMER_ID=$(tofu state show apisix_consumer.$resource 2>&1 | grep '^\s*username\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    if [ -z "$CONSUMER_ID" ]; then
        log_error "Failed to get consumer ID for $resource"
        tofu state show apisix_consumer.$resource 2>&1 | head -5
        exit 1
    fi
    log_info "Consumer '$resource' created with username: $CONSUMER_ID"
    
    # Verify via APISIX API
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/consumers/$CONSUMER_ID" \
        -H "X-API-KEY: test123456789")
    if [ "$RESPONSE" != "200" ]; then
        log_error "Consumer '$resource' not found in APISIX (HTTP $RESPONSE)"
        curl -s "http://localhost:9180/apisix/admin/consumers/$CONSUMER_ID" -H "X-API-KEY: test123456789" | head -20
        exit 1
    fi
done
log_info "✓ All consumers verified in APISIX API"

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

# Test 3: Verify consumer configurations via API
log_info "Test 3: Verify consumer configurations"

# Verify key_auth consumer
KEY_AUTH_ID=$(tofu state show apisix_consumer.key_auth 2>/dev/null | grep '^\s*username\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/consumers/$KEY_AUTH_ID" -H "X-API-KEY: test123456789")
PLUGINS_COUNT=$(echo "$RESPONSE" | jq -r '.value.plugins | keys | length')
[ "$PLUGINS_COUNT" = "1" ] || { log_error "key_auth consumer plugins mismatch: got $PLUGINS_COUNT"; exit 1; }

# Verify jwt_auth consumer
JWT_AUTH_ID=$(tofu state show apisix_consumer.jwt_auth 2>/dev/null | grep '^\s*username\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/consumers/$JWT_AUTH_ID" -H "X-API-KEY: test123456789")
JWT_KEY=$(echo "$RESPONSE" | jq -r '.value.plugins["jwt-auth"].key')
[ "$JWT_KEY" = "jwt-test-key" ] || { log_error "jwt_auth consumer key mismatch: got $JWT_KEY"; exit 1; }

# Verify with_labels consumer
WITH_LABELS_ID=$(tofu state show apisix_consumer.with_labels 2>/dev/null | grep '^\s*username\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/consumers/$WITH_LABELS_ID" -H "X-API-KEY: test123456789")
LABELS_COUNT=$(echo "$RESPONSE" | jq -r '.value.labels | keys | length')
[ "$LABELS_COUNT" = "3" ] || { log_error "with_labels consumer labels mismatch: got $LABELS_COUNT"; exit 1; }

# Verify with_group consumer
WITH_GROUP_ID=$(tofu state show apisix_consumer.with_group 2>/dev/null | grep '^\s*username\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/consumers/$WITH_GROUP_ID" -H "X-API-KEY: test123456789")
GROUP_ID=$(echo "$RESPONSE" | jq -r '.value.group_id')
[ "$GROUP_ID" = "test-consumer-group" ] || { log_error "with_group consumer group_id mismatch: got $GROUP_ID"; exit 1; }

log_info "✓ Consumer configurations verified"

# Test 4: Destroy all consumers
log_info "Test 4: Destroy consumers"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

# Verify all consumers were deleted
for resource in basic key_auth jwt_auth with_labels hmac_auth with_group; do
    CONSUMER_ID=$(tofu state show apisix_consumer.$resource 2>/dev/null | grep "^ *username *" | cut -d'"' -f2 || echo "")
    if [ -n "$CONSUMER_ID" ]; then
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/consumers/$CONSUMER_ID" \
            -H "X-API-KEY: test123456789")
        if [ "$RESPONSE" != "404" ]; then
            log_error "Consumer '$resource' still exists in APISIX (HTTP $RESPONSE)"
            exit 1
        fi
    fi
done
log_info "✓ All consumers deleted successfully"

# Test 5: Recreate all consumers
log_info "Test 5: Recreate consumers"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

for resource in basic key_auth jwt_auth with_labels hmac_auth with_group; do
    CONSUMER_ID=$(tofu state show apisix_consumer.$resource 2>/dev/null | grep "^ *username *" | cut -d'"' -f2)
    if [ -z "$CONSUMER_ID" ]; then
        log_error "Failed to get consumer ID for $resource after recreation"
        exit 1
    fi
done
log_info "✓ All consumers recreated successfully"

# Test 6: Import test for all consumers
log_info "Test 6: Import test"
for resource in basic key_auth jwt_auth with_labels hmac_auth with_group; do
    CONSUMER_ID=$(tofu state show apisix_consumer.$resource 2>/dev/null | grep '^\s*username\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    
    # Remove from state
    echo "Executing: tofu state rm apisix_consumer.$resource"
    tofu state rm apisix_consumer.$resource
    
    # Import back
    echo "Executing: tofu import apisix_consumer.$resource $CONSUMER_ID"
    tofu import apisix_consumer.$resource "$CONSUMER_ID"
    
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
