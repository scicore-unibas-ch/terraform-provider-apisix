#!/bin/bash
set -e

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
tofu plan -detailed-exitcode -lock=false || exit 1
tofu destroy -auto-approve -lock=false

echo "=== Cycle 2: Create → Import → Verify → Destroy ==="
tofu apply -auto-approve -lock=false
tofu plan -detailed-exitcode -lock=false || exit 1

# Import test
for resource in basic advanced with_vars complete with_script; do
    ID=$(tofu state show apisix_route.$resource 2>/dev/null | grep '^\s*id\s*=\|^\s*username\s*=\|^\s*rule_id\s*=\|^\s*config_id\s*=' | head -1 | sed 's/.*= *"\([^"]*\)".*/\1/')
    tofu state rm apisix_route.$resource
    tofu import apisix_route.$resource "$ID"
done

tofu apply -auto-approve -lock=false
tofu plan -detailed-exitcode -lock=false || exit 1
tofu destroy -auto-approve -lock=false

# Cleanup
rm -f .tofurc
echo "✓ All route tests passed"
