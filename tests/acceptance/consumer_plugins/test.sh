#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$TEST_DIR"

ADMIN="${APISIX_BASE_URL:-http://localhost:9180/apisix/admin}"
KEY="${APISIX_ADMIN_KEY:-test123456789}"

BASIC_CONSUMER="test-consumer-plugins-basic"
MULTI_CONSUMER="test-consumer-plugins-multi"
CREDENTIAL="test-consumer-plugins-key"

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

api() { # api <method> <path> [body]
    if [ $# -ge 3 ]; then
        curl -s -X "$1" "$ADMIN$2" -H "X-API-KEY: $KEY" -H 'Content-Type: application/json' -d "$3"
    else
        curl -s -X "$1" "$ADMIN$2" -H "X-API-KEY: $KEY"
    fi
}

# The consumers are owned by an external system, not by Terraform. Create them
# the way that system would, with a credential Terraform must never touch.
create_consumers() {
    for c in "$BASIC_CONSUMER" "$MULTI_CONSUMER"; do
        api PUT "/consumers/$c" \
            "{\"username\":\"$c\",\"plugins\":{\"key-auth\":{\"key\":\"$CREDENTIAL-$c\"}},\"labels\":{\"owner\":\"external\"}}" \
            > /dev/null
    done
}

delete_consumers() {
    for c in "$BASIC_CONSUMER" "$MULTI_CONSUMER"; do
        api DELETE "/consumers/$c" > /dev/null || true
    done
}

# Parse the response instead of grepping it. The Admin API wraps the object as
# {"value":{...},"key":"/apisix/consumers/<name>",...}, so a body carrying a
# key-auth credential contains TWO "key" fields and their order is not stable
# across APISIX versions — grep + head picked the etcd path on some of them.
# consumer_field <consumer> <python expression over the unwrapped object>
consumer_field() {
    api GET "/consumers/$1" | python3 -c "
import json, sys
obj = json.load(sys.stdin)
obj = obj.get('value', obj)
print($2)
"
}

# assert_credential_intact <consumer>
assert_credential_intact() {
    local consumer="$1"
    local got
    got=$(consumer_field "$consumer" "obj.get('plugins', {}).get('key-auth', {}).get('key', '')")
    if [ "$got" != "$CREDENTIAL-$consumer" ]; then
        echo "✗ credential on $consumer was modified: got '$got', want '$CREDENTIAL-$consumer'"
        exit 1
    fi
    echo "  ✓ key-auth intact on $consumer"
}

# assert_plugin <consumer> <plugin> <present|absent>
assert_plugin() {
    local consumer="$1" plugin="$2" want="$3" got
    got=$(consumer_field "$consumer" "'present' if '$plugin' in obj.get('plugins', {}) else 'absent'")
    if [ "$got" != "$want" ]; then
        echo "✗ $plugin is $got on $consumer, expected $want"
        exit 1
    fi
    echo "  ✓ $plugin $want on $consumer"
}

cleanup() {
    tofu destroy -auto-approve -lock=false > /dev/null 2>&1 || true
    delete_consumers
    rm -f .tofurc
}
trap cleanup EXIT

delete_consumers
create_consumers

echo "=== Cycle 1: Create → Verify Idempotency → Destroy ==="
tofu apply -auto-approve -lock=false
tofu plan -detailed-exitcode -lock=false
assert_plugin "$BASIC_CONSUMER" "limit-count" present
assert_plugin "$MULTI_CONSUMER" "limit-req" present
assert_credential_intact "$BASIC_CONSUMER"
assert_credential_intact "$MULTI_CONSUMER"

tofu destroy -auto-approve -lock=false
# Destroy must detach the managed plugins and leave the consumer + credential.
assert_plugin "$BASIC_CONSUMER" "limit-count" absent
assert_credential_intact "$BASIC_CONSUMER"
assert_credential_intact "$MULTI_CONSUMER"

echo "=== Cycle 2: Create → Import → Verify → Destroy ==="
tofu apply -auto-approve -lock=false
tofu plan -detailed-exitcode -lock=false

# Import IDs are "<consumer>/<plugin>[,<plugin>...]" — importing by bare
# consumer name is refused so that key-auth can never be pulled into state.
tofu state rm apisix_consumer_plugins.basic
tofu import apisix_consumer_plugins.basic "$BASIC_CONSUMER/limit-count"

tofu state rm apisix_consumer_plugins.multi
tofu import apisix_consumer_plugins.multi "$MULTI_CONSUMER/limit-count,limit-req"

tofu plan -detailed-exitcode -lock=false

if tofu import apisix_consumer_plugins.basic "$BASIC_CONSUMER" > /dev/null 2>&1; then
    echo "✗ import by bare consumer name should have been rejected"
    exit 1
fi
echo "  ✓ import without plugin names rejected"

echo "=== Cycle 3: External rewrite → re-apply restores the plugin ==="
# Simulate the owning system rotating the credential: a full PUT that drops
# our plugin. Re-applying must restore it on top of the NEW credential.
api PUT "/consumers/$BASIC_CONSUMER" \
    "{\"username\":\"$BASIC_CONSUMER\",\"plugins\":{\"key-auth\":{\"key\":\"$CREDENTIAL-$BASIC_CONSUMER\"}}}" > /dev/null
assert_plugin "$BASIC_CONSUMER" "limit-count" absent

tofu apply -auto-approve -lock=false
assert_plugin "$BASIC_CONSUMER" "limit-count" present
assert_credential_intact "$BASIC_CONSUMER"

tofu destroy -auto-approve -lock=false
assert_credential_intact "$BASIC_CONSUMER"

echo "✓ All consumer_plugins tests passed"
