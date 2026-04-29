# GitHub Actions Setup Guide

## Overview

This provider uses GitHub Actions for:
1. **Unit Tests** - Run on every push and pull request
2. **Acceptance Tests** - Run on main branch pushes and PRs (with Docker Compose)
3. **Releases** - Publish to GitHub Releases and OpenTofu Registry on tag push

## Workflows

### 1. Unit Tests (`.github/workflows/unit-tests.yml`)

Runs on every push and pull request:
- Sets up Go environment
- Runs `go vet` for code quality
- Runs all unit tests with verbose output

**Triggers:**
- Push to any branch
- Pull requests

### 2. Acceptance Tests (`.github/workflows/acceptance-tests.yml`)

Runs full acceptance test suite with real APISIX instance:
- Starts APISIX + etcd via Docker Compose
- Builds provider binary
- Runs acceptance tests for all 7 resources:
  - upstream (6 tests)
  - route (8 tests)
  - service (6 tests)
  - consumer (6 tests)
  - consumer_group (6 tests)
  - plugin_config (6 tests)
  - global_rule (6 tests)
- SSL tests are skipped (requires SSL proxy configuration)

**Triggers:**
- Push to main branch
- Pull requests to main
- Manual workflow dispatch

**Note:** SSL acceptance tests are not run in CI due to SSL proxy configuration complexity. The resource is fully implemented with unit tests and can be tested manually.

### 3. Release (`.github/workflows/release.yml`)

Publishes releases to GitHub and prepares for OpenTofu Registry:
- Builds binaries for linux/darwin (amd64/arm64)
- Creates checksums
- Signs with GPG key
- Generates changelog from git tags

**Triggers:**
- Push of version tags (e.g., `v0.1.0`, `v1.0.0`)

## Setup Instructions

### 1. Enable GitHub Actions

GitHub Actions are enabled by default for all repositories.

### 2. Configure GPG Key for Signing Releases

To sign release artifacts:

1. **Generate GPG key** (if you don't have one):
   ```bash
   gpg --full-generate-key
   ```

2. **Export the private key**:
   ```bash
   gpg --armor --export-secret-key YOUR_KEY_ID
   ```

3. **Get the fingerprint**:
   ```bash
   gpg --list-keys --keyid-format LONG
   ```

4. **Add GitHub Secrets**:
   - Go to: `https://github.com/YOUR_USERNAME/terraform-provider-apisix/settings/secrets/actions`
   - Add `GPG_PRIVATE_KEY`: Paste the armored private key
   - Add `GPG_PASSPHRASE`: Your GPG key passphrase

### 3. Test Workflows Locally (Optional)

You can test the provider locally using the provided Makefile:

```bash
# Run unit tests
make test

# Run acceptance tests (requires Docker Compose)
cd tests/
docker compose up -d  # Start APISIX cluster

# Then run individual test suites:
cd acceptance/upstream && ./test.sh
cd ../route && ./test.sh
cd ../service && ./test.sh
cd ../consumer && ./test.sh
cd ../consumer_group && ./test.sh
cd ../plugin_config && ./test.sh
cd ../global_rule && ./test.sh

# Cleanup when done
docker compose down
```

**Note:** The acceptance tests workflow in GitHub Actions uses Docker Compose automatically, so tests that pass locally should pass in CI.

## OpenTofu Registry Publication

After the first release is published to GitHub:

1. **Verify release on GitHub**:
   - Check that binaries, checksums, and signatures are present
   - Verify changelog is correct

2. **Submit to OpenTofu Registry**:
   - Go to: https://registry.opentofu.org/
   - Click "Publish"
   - Enter repository URL: `https://github.com/scicore-unibas-ch/terraform-provider-apisix`
   - Follow the verification steps

3. **Registry Requirements**:
   - Repository must be public
   - Releases must be signed with GPG
   - Provider must follow naming convention: `terraform-provider-<name>`
   - Documentation must be in `docs/` directory
   - Examples must be in `examples/` directory

## Workflow Status

Check workflow status at:
- **Actions Tab**: `https://github.com/YOUR_USERNAME/terraform-provider-apisix/actions`

## Troubleshooting

### Acceptance Tests Failing

If acceptance tests fail in CI:

1. Check Docker Compose startup logs
2. Verify APISIX is ready (curl to Admin API)
3. Check if test cleanup from previous runs completed

### Release Workflow Failing

Common issues:

1. **GPG key not configured**: Add secrets as described above
2. **Tag format incorrect**: Must be `v*` (e.g., `v0.1.0`)
3. **Go build errors**: Run `go build` locally first

### Manual Workflow Trigger

You can manually trigger acceptance tests:
1. Go to Actions tab
2. Select "Acceptance Tests" workflow
3. Click "Run workflow"
4. Select branch and click "Run workflow"

## Next Steps

After workflows are working:

1. ✅ Verify unit tests pass in CI
2. ✅ Verify acceptance tests pass in CI
3. ✅ Create first release tag: `git tag v0.1.0 && git push origin v0.1.0`
4. ✅ Verify release workflow completes successfully
5. ✅ Submit provider to OpenTofu Registry
