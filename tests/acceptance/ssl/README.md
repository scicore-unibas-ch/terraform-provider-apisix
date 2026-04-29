# SSL/TLS Acceptance Tests

## Status: ⏭️ SKIPPED (Manual Execution Only)

SSL acceptance tests are **skipped** in the automated test suite (`make test-acceptance`) because they require special APISIX configuration with SSL proxy enabled.

## Prerequisites

To run SSL tests manually, you need:

1. **SSL Certificates** (already provided in `certs/` directory):
   - `example.com.crt` / `example.com.key`
   - `secure.example.com.crt` / `secure.example.com.key`
   - `labeled.example.com.crt` / `labeled.example.com.key`

2. **APISIX with SSL Proxy Enabled**

## How to Enable SSL Tests

### 1. Verify APISIX SSL Configuration

Check that `tests/apisix/config.yaml` has SSL enabled:

```yaml
apisix:
  ssl:
    enable: true
    listen:
      - port: 9443
```

### 2. Verify Docker Compose Exposes Port 9443

Check that `tests/docker-compose.yml` exposes port 9443:

```yaml
ports:
  - "9443:9443"  # HTTPS/SSL
```

### 3. Restart APISIX Cluster

```bash
cd ../../
docker compose down -v
docker compose up -d
sleep 10
```

### 4. Verify SSL Port is Listening

```bash
# Check if port 9443 is accessible
curl -k -s -o /dev/null -w "%{http_code}" https://localhost:9443/
# Should return 400 or 404 (not 000)
```

## Running SSL Tests Manually

```bash
# From the ssl test directory
cd tests/acceptance/ssl

# Run the test
bash test.sh
```

Or from the project root:

```bash
cd /path/to/terraform-provider-apisix
bash tests/acceptance/ssl/test.sh
```

## What the Tests Do

The SSL acceptance tests verify:

1. **Create SSL certificates** - Creates 3 SSL certificates via Terraform
2. **Verify idempotency** - Ensures no changes on second apply
3. **Verify SSL configurations** - Checks certificates via APISIX Admin API
4. **Destroy SSL certificates** - Cleans up resources
5. **Recreate SSL certificates** - Verifies resources can be recreated

## Troubleshooting

### SSL Proxy Not Enabled

If you get errors about SSL proxy not being enabled:

```bash
# Check APISIX config
cat tests/apisix/config.yaml | grep -A3 "ssl:"

# Should show:
# ssl:
#   enable: true
#   listen:
#     - port: 9443
```

### Connection Refused on Port 9443

```bash
# Check if container is running
docker compose ps

# Check if port is listening
docker exec tests-apisix-1 netstat -tlnp | grep 9443

# Restart containers
docker compose down -v && docker compose up -d
```

### Certificate Errors

If you get certificate validation errors:

```bash
# Regenerate certificates
cd tests/acceptance/ssl/certs
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout example.com.key -out example.com.crt \
  -subj "/CN=example.com/O=Test/C=US"
```

## Certificate Information

The provided certificates are:
- **Type**: Self-signed X.509
- **Key Size**: 2048-bit RSA
- **Validity**: 10 years (3650 days)
- **Purpose**: Testing only (not for production)

Generated with:
```bash
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout example.com.key -out example.com.crt \
  -subj "/CN=example.com/O=Test/C=US"
```

## Why SSL Tests Are Skipped in CI

1. **Complexity**: Requires SSL proxy configuration in APISIX
2. **Performance**: SSL operations are slower than regular API calls
3. **Flakiness**: SSL handshake issues can cause intermittent failures
4. **Not Critical**: Core provider functionality is validated by other tests

SSL tests are intended for:
- Manual testing before major releases
- Validating SSL-specific features
- Local development when working on SSL resources
