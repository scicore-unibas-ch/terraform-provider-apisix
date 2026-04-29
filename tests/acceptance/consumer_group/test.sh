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
docker compose down --volumes --remove-orphans >/dev/null 2>&1 || true
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


# Test 1: Create all consumer groups
log_info "Test 1: Create consumer groups (basic, with_plugins, multi_plugins, with_name, with_labels, consumer_test)"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

# Verify all consumer groups were created
for resource in basic with_plugins multi_plugins with_name with_labels consumer_test; do
    GROUP_ID=$(tofu state show apisix_consumer_group.$resource 2>&1 | grep '^\s*group_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    if [ -z "$GROUP_ID" ]; then
        log_error "Failed to get consumer group ID for $resource"
        tofu state show apisix_consumer_group.$resource 2>&1 | head -5
        exit 1
    fi
    log_info "Consumer group '$resource' created with ID: $GROUP_ID"
    
    # Verify via APISIX API
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/consumer_groups/$GROUP_ID" \
        -H "X-API-KEY: test123456789")
    if [ "$RESPONSE" != "200" ]; then
        log_error "Consumer group '$resource' not found in APISIX (HTTP $RESPONSE)"
        curl -s "http://localhost:9180/apisix/admin/consumer_groups/$GROUP_ID" -H "X-API-KEY: test123456789" | head -20
        exit 1
    fi
done
log_info "✓ All consumer groups verified in APISIX API"

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

# Test 3: Verify consumer group configurations via API
log_info "Test 3: Verify consumer group configurations"

# Verify with_plugins consumer group
WITH_PLUGINS_ID=$(tofu state show apisix_consumer_group.with_plugins 2>/dev/null | grep '^\s*group_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/consumer_groups/$WITH_PLUGINS_ID" -H "X-API-KEY: test123456789")
PLUGINS_COUNT=$(echo "$RESPONSE" | jq -r '.value.plugins | keys | length')
[ "$PLUGINS_COUNT" = "1" ] || { log_error "with_plugins consumer group plugins mismatch: got $PLUGINS_COUNT"; exit 1; }

# Verify multi_plugins consumer group
MULTI_PLUGINS_ID=$(tofu state show apisix_consumer_group.multi_plugins 2>/dev/null | grep '^\s*group_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/consumer_groups/$MULTI_PLUGINS_ID" -H "X-API-KEY: test123456789")
PLUGINS_COUNT=$(echo "$RESPONSE" | jq -r '.value.plugins | keys | length')
[ "$PLUGINS_COUNT" = "2" ] || { log_error "multi_plugins consumer group plugins mismatch: got $PLUGINS_COUNT"; exit 1; }

# Verify with_name consumer group
WITH_NAME_ID=$(tofu state show apisix_consumer_group.with_name 2>/dev/null | grep '^\s*group_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/consumer_groups/$WITH_NAME_ID" -H "X-API-KEY: test123456789")
NAME=$(echo "$RESPONSE" | jq -r '.value.name')
[ "$NAME" = "Premium Tier Group" ] || { log_error "with_name consumer group name mismatch: got $NAME"; exit 1; }

# Verify with_labels consumer group
WITH_LABELS_ID=$(tofu state show apisix_consumer_group.with_labels 2>/dev/null | grep '^\s*group_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -s "http://localhost:9180/apisix/admin/consumer_groups/$WITH_LABELS_ID" -H "X-API-KEY: test123456789")
LABELS_COUNT=$(echo "$RESPONSE" | jq -r '.value.labels | keys | length')
[ "$LABELS_COUNT" = "3" ] || { log_error "with_labels consumer group labels mismatch: got $LABELS_COUNT"; exit 1; }

log_info "✓ Consumer group configurations verified"

# Test 4: Destroy all consumer groups
log_info "Test 4: Destroy consumer groups"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

# Verify all consumer groups were deleted
for resource in basic with_plugins multi_plugins with_name with_labels consumer_test; do
    GROUP_ID=$(tofu state show apisix_consumer_group.$resource 2>/dev/null | grep "^ *group_id *" | cut -d'"' -f2 || echo "")
    if [ -n "$GROUP_ID" ]; then
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/consumer_groups/$GROUP_ID" \
            -H "X-API-KEY: test123456789")
        if [ "$RESPONSE" != "404" ]; then
            log_error "Consumer group '$resource' still exists in APISIX (HTTP $RESPONSE)"
            exit 1
        fi
    fi
done
log_info "✓ All consumer groups deleted successfully"

# Test 5: Recreate all consumer groups
log_info "Test 5: Recreate consumer groups"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

for resource in basic with_plugins multi_plugins with_name with_labels consumer_test; do
    GROUP_ID=$(tofu state show apisix_consumer_group.$resource 2>/dev/null | grep "^ *group_id *" | cut -d'"' -f2)
    if [ -z "$GROUP_ID" ]; then
        log_error "Failed to get consumer group ID for $resource after recreation"
        exit 1
    fi
done
log_info "✓ All consumer groups recreated successfully"

# Test 6: Import test for all consumer groups
log_info "Test 6: Import test"
for resource in basic with_plugins multi_plugins with_name with_labels consumer_test; do
    GROUP_ID=$(tofu state show apisix_consumer_group.$resource 2>/dev/null | grep '^\s*group_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    
    # Remove from state
    echo "Executing: tofu state rm apisix_consumer_group.$resource"
    tofu state rm apisix_consumer_group.$resource
    
    # Import back
    echo "Executing: tofu import apisix_consumer_group.$resource $GROUP_ID"
    tofu import apisix_consumer_group.$resource "$GROUP_ID"
    
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
