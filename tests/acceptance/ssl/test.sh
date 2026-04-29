#!/bin/bash
# SSL acceptance tests - SKIPPED
# These tests require SSL certificates and special APISIX configuration
# They should be run manually when needed with proper SSL setup

echo "⚠ SSL acceptance tests SKIPPED"
echo "SSL tests require:"
echo "  1. Valid SSL certificates in tests/apisix/ssl/"
echo "  2. APISIX configured with SSL proxy enabled"
echo "  3. Port 9443 accessible"
echo ""
echo "To enable SSL tests:"
echo "  1. Generate SSL certificates"
echo "  2. Update tests/apisix/config.yaml with ssl.enable: true"
echo "  3. Run tests manually"
echo ""
echo "Skipping SSL tests..."
exit 0
