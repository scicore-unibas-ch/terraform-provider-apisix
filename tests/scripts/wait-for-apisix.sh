#!/bin/bash

# Wait for APISIX Admin API to be ready
# Timeout: 60 seconds

MAX_ATTEMPTS=30
ATTEMPT=0

echo "Waiting for APISIX Admin API to be ready..."

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:9180/apisix/admin/routes | grep -q "200\|401\|403"; then
        echo "✓ APISIX Admin API is ready"
        exit 0
    fi
    
    ATTEMPT=$((ATTEMPT + 1))
    echo "  Attempt $ATTEMPT/$MAX_ATTEMPTS - waiting..."
    sleep 2
done

echo "✗ Timeout waiting for APISIX Admin API"
exit 1
