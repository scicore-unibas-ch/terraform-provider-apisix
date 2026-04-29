#!/bin/bash
set -e

# Configuration
CLEANUP_ON_FAILURE=${CLEANUP_ON_FAILURE:-true}
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$TEST_DIR"

# Generate temporary .tofurc for this test
log_info "Generating temporary provider config..."
cat > .tofurc << TOFURC
provider_installation {
  dev_overrides {
    "scicore-unibas-ch/apisix" = "/home/escobar/github/terraform-provider-apisix"
  }
  direct {}
}
TOFURC
export TF_CLI_CONFIG_FILE="$TEST_DIR/.tofurc"

# Generate temporary .tofurc for this test
log_info "Generating temporary provider config..."
cat > .tofurc << TOFURC
provider_installation {
  dev_overrides {
    "scicore-unibas-ch/apisix" = "/home/escobar/github/terraform-provider-apisix"
  }
  direct {}
}
TOFURC
export TF_CLI_CONFIG_FILE="$TEST_DIR/.tofurc"

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
        for id in example.com labeled.example.com secure.example.com; do curl -s -X DELETE "http://localhost:9180/apisix/admin/routesssls/$id" -H "X-API-KEY: test123456789" > /dev/null 2>&1 || true; done
    else
        log_warn "Leaving resources for debugging (set CLEANUP_ON_FAILURE=true to auto-cleanup)"
    fi
}

trap cleanup EXIT

# Check if SSL port is available
log_info "Checking if APISIX SSL proxy is enabled (port 9443)..."
if ! curl -k -s -o /dev/null -w "%{http_code}" "https://localhost:9443" --connect-timeout 5 2>/dev/null | grep -q "404\|400"; then
    log_error "APISIX SSL proxy is not enabled on port 9443"
    log_error "Please ensure:"
    log_error "  1. tests/apisix/config.yaml has 'apisix.ssl.enable: true'"
    log_error "  2. tests/docker-compose.yml exposes port 9443"
    log_error "  3. Docker containers are restarted: docker-compose down && docker-compose up -d"
    exit 1
fi
log_info "✓ APISIX SSL proxy is enabled"

# Initialize
log_info "Initializing Terraform..."
# echo "Executing: tofu init -input=false"
# tofu init -input=false

# Remove lock files for clean test
rm -f .terraform.lock.hcl .tofurc 2>/dev/null || true

# Remove lock files for clean test
rm -f .terraform.lock.hcl .tofurc 2>/dev/null || true

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
    for id in example.com labeled.example.com secure.example.com; do curl -s -X DELETE "http://localhost:9180/apisix/admin/routesssls/$id" -H "X-API-KEY: test123456789" > /dev/null 2>&1 || true; done


# Test 1: Create all SSL certificates
log_info "Test 1: Create SSL certificates (basic, multi_sni, tls13, with_labels)"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

# Verify all SSL certificates were created
for resource in basic multi_sni tls13 with_labels; do
    SSL_ID=$(tofu state show apisix_ssl.$resource 2>&1 | grep '^\s*sni\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    if [ -z "$SSL_ID" ]; then
        log_error "Failed to get SSL SNI for $resource"
        tofu state show apisix_ssl.$resource 2>&1 | head -5
        exit 1
    fi
    log_info "SSL certificate '$resource' created with SNI: $SSL_ID"
    
    # Verify via APISIX API
    RESPONSE=$(curl -k -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/routesssl/$SSL_ID" \
        -H "X-API-KEY: test123456789")
    if [ "$RESPONSE" != "200" ]; then
        log_error "SSL certificate '$resource' not found in APISIX (HTTP $RESPONSE)"
        curl -k -s "http://localhost:9180/apisix/admin/routesssl/$SSL_ID" -H "X-API-KEY: test123456789" | head -20
        exit 1
    fi
done
log_info "✓ All SSL certificates verified in APISIX API"

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

# Test 3: Verify SSL configurations via API
log_info "Test 3: Verify SSL configurations"

# Verify multi_sni certificate
MULTI_SNI_ID=$(tofu state show apisix_ssl.multi_sni 2>/dev/null | grep '^\s*sni\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -k -s "http://localhost:9180/apisix/admin/routesssl/$MULTI_SNI_ID" -H "X-API-KEY: test123456789")
SNIS_COUNT=$(echo "$RESPONSE" | jq -r '.value.snis | length')
[ "$SNIS_COUNT" = "2" ] || { log_error "multi_sni certificate SNIs mismatch: got $SNIS_COUNT"; exit 1; }

# Verify tls13 certificate
TLS13_ID=$(tofu state show apisix_ssl.tls13 2>/dev/null | grep '^\s*sni\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -k -s "http://localhost:9180/apisix/admin/routesssl/$TLS13_ID" -H "X-API-KEY: test123456789")
SSL_PROTOCOLS=$(echo "$RESPONSE" | jq -r '.value.ssl_protocols | length')
[ "$SSL_PROTOCOLS" = "1" ] || { log_error "tls13 certificate ssl_protocols mismatch: got $SSL_PROTOCOLS"; exit 1; }
PROTOCOL=$(echo "$RESPONSE" | jq -r '.value.ssl_protocols[0]')
[ "$PROTOCOL" = "TLSv1.3" ] || { log_error "tls13 certificate protocol mismatch: got $PROTOCOL"; exit 1; }

# Verify with_labels certificate
WITH_LABELS_ID=$(tofu state show apisix_ssl.with_labels 2>/dev/null | grep '^\s*sni\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
RESPONSE=$(curl -k -s "http://localhost:9180/apisix/admin/routesssl/$WITH_LABELS_ID" -H "X-API-KEY: test123456789")
LABELS_COUNT=$(echo "$RESPONSE" | jq -r '.value.labels | keys | length')
[ "$LABELS_COUNT" = "3" ] || { log_error "with_labels certificate labels mismatch: got $LABELS_COUNT"; exit 1; }

log_info "✓ SSL configurations verified"

# Test 4: Destroy all SSL certificates
log_info "Test 4: Destroy SSL certificates"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

# Verify all SSL certificates were deleted
for resource in basic multi_sni tls13 with_labels; do
    SSL_ID=$(tofu state show apisix_ssl.$resource 2>/dev/null | grep "^ *sni *" | cut -d'"' -f2 || echo "")
    if [ -n "$SSL_ID" ]; then
        RESPONSE=$(curl -k -s -o /dev/null -w "%{http_code}" "http://localhost:9180/apisix/admin/routesssl/$SSL_ID" \
            -H "X-API-KEY: test123456789")
        if [ "$RESPONSE" != "404" ]; then
            log_error "SSL certificate '$resource' still exists in APISIX (HTTP $RESPONSE)"
            exit 1
        fi
    fi
done
log_info "✓ All SSL certificates deleted successfully"

# Test 5: Recreate all SSL certificates
log_info "Test 5: Recreate SSL certificates"
echo "Executing: tofu apply -auto-approve -lock=false"
tofu apply -auto-approve -lock=false

for resource in basic multi_sni tls13 with_labels; do
    SSL_ID=$(tofu state show apisix_ssl.$resource 2>/dev/null | grep "^ *sni *" | cut -d'"' -f2)
    if [ -z "$SSL_ID" ]; then
        log_error "Failed to get SSL SNI for $resource after recreation"
        exit 1
    fi
done
log_info "✓ All SSL certificates recreated successfully"

# Test 6: Import test for all SSL certificates
log_info "Test 6: Import test"
for resource in basic multi_sni tls13 with_labels; do
    SSL_ID=$(tofu state show apisix_ssl.$resource 2>/dev/null | grep '^\s*sni\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    
    # Remove from state
    echo "Executing: tofu state rm apisix_ssl.$resource"
    tofu state rm apisix_ssl.$resource
    
    # Import back
    echo "Executing: tofu import apisix_ssl.$resource $SSL_ID"
    tofu import apisix_ssl.$resource "$SSL_ID"
    
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
done

# Final cleanup
log_info "Cleanup"
echo "Executing: tofu destroy -auto-approve -lock=false"
tofu destroy -auto-approve -lock=false

log_info "✓ All acceptance tests passed!"
