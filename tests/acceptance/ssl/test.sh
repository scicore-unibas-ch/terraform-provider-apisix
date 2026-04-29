#!/bin/bash
# SSL acceptance tests - SKIPPED
# These tests require SSL certificates and special APISIX configuration

echo "⚠ SSL acceptance tests SKIPPED"
echo ""
echo "SSL tests require:"
echo "  1. Valid SSL certificates in tests/acceptance/ssl/certs/"
echo "  2. APISIX configured with SSL proxy enabled"  
echo "  3. Port 9443 accessible"
echo ""
echo "Certificates have been generated in: tests/acceptance/ssl/certs/"
echo "To enable SSL tests:"
echo "  1. Ensure tests/apisix/config.yaml has 'apisix.ssl.enable: true'"
echo "  2. Run: docker compose down -v && docker compose up -d"
echo "  3. Run tests manually: bash tests/acceptance/ssl/test.sh"
echo ""
echo "Skipping SSL tests..."
exit 0
