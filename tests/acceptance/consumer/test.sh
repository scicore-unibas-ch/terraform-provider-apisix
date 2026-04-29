#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$TEST_DIR"

# Generate .tofurc
PROVIDER_DIR="$(cd "$TEST_DIR/../../.." && pwd)"
cat > .tofurc << TOFURC
provider_installation {
  dev_overrides {
    "scicore-unibas-ch/apisix" = "$PROVIDER_DIR"
  }
  direct {}
}
TOFURC
export TF_CLI_CONFIG_FILE="$TEST_DIR/.tofurc"

echo "=== Cycle 1: Create → Verify Idempotency → Destroy ==="
tofu apply -auto-approve -lock=false
tofu plan -detailed-exitcode -lock=false
tofu destroy -auto-approve -lock=false

echo "=== Cycle 2: Create → Import → Verify → Destroy ==="
tofu apply -auto-approve -lock=false
tofu plan -detailed-exitcode -lock=false

# Import test
for resource in basic key_auth jwt_auth with_labels hmac_auth with_group; do
    ID=$(tofu state show apisix_consumer.$resource | grep -E '^\s*(id|username|rule_id|config_id)\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    tofu state rm apisix_consumer.$resource
    tofu import apisix_consumer.$resource "$ID"
done

tofu apply -auto-approve -lock=false
tofu plan -detailed-exitcode -lock=false
tofu destroy -auto-approve -lock=false

# Cleanup
rm -f .tofurc
echo "✓ All consumer tests passed"
