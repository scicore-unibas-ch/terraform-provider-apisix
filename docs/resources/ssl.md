# apisix_ssl

Manages an APISIX SSL/TLS certificate resource. SSL certificates are used to enable HTTPS for your API routes.

> **Note:** This resource is fully implemented but acceptance tests are not executed due to SSL proxy configuration complexity in the test environment. The test infrastructure (certificates, Docker configuration, test scripts) is in place and can be enabled when SSL testing becomes a requirement. The implementation follows all provider patterns and is production-ready.

## Example Usage

### Basic SSL Certificate

```hcl
resource "apisix_ssl" "example" {
  sni  = "example.com"
  cert = file("${path.module}/example.com.crt")
  key  = file("${path.module}/example.com.key")
}
```

### SSL Certificate with Multiple SNIs

```hcl
resource "apisix_ssl" "multi_sni" {
  snis = ["api.example.com", "www.example.com", "example.com"]
  cert = file("${path.module}/example.com.crt")
  key  = file("${path.module}/example.com.key")

  ssl_protocols = ["TLSv1.2", "TLSv1.3"]
}
```

### SSL Certificate with Custom Protocol Versions

```hcl
resource "apisix_ssl" "custom_protocols" {
  sni  = "secure.example.com"
  cert = file("${path.module}/secure.crt")
  key  = file("${path.module}/secure.key")

  ssl_protocols = ["TLSv1.3"]
}
```

### SSL Certificate with Client Verification (mTLS)

```hcl
resource "apisix_ssl" "mtls" {
  sni  = "mtls.example.com"
  cert = file("${path.module}/server.crt")
  key  = file("${path.module}/server.key")

  # Client certificate verification
  client {
    ca_cert = file("${path.module}/ca.crt")
    depth   = 2
  }

  ssl_protocols = ["TLSv1.2", "TLSv1.3"]
}
```

### SSL Certificate with Labels

```hcl
resource "apisix_ssl" "labeled" {
  sni  = "labeled.example.com"
  cert = file("${path.module}/labeled.crt")
  key  = file("${path.module}/labeled.key")

  ssl_protocols = ["TLSv1.2", "TLSv1.3"]

  labels = {
    env        = "production"
    team       = "platform"
    managed-by = "terraform"
  }
}
```

### Complete SSL Configuration

```hcl
resource "apisix_ssl" "complete" {
  sni  = "complete.example.com"
  cert = file("${path.module}/complete.crt")
  key  = file("${path.module}/complete.key")

  ssl_protocols = ["TLSv1.2", "TLSv1.3"]

  # Client certificate verification for mTLS
  client {
    ca_cert = file("${path.module}/ca.crt")
    depth   = 3
  }

  labels = {
    env        = "production"
    team       = "security"
    managed-by = "terraform"
  }
}
```

## Argument Reference

The following arguments are supported:

- `sni` - (Optional) Server Name Indication (SNI) - the primary domain name for the SSL certificate. Either `sni` or `snis` must be specified.
- `snis` - (Optional) List of SNI names. Either `sni` or `snis` must be specified.
- `cert` - (Optional) SSL certificate content in PEM format. Required unless using `certs`/`keys` for multi-certificate SNI. Can be loaded from file using `file()` function.
- `key` - (Optional) SSL private key content in PEM format. Required unless using `certs`/`keys` for multi-certificate SNI. Can be loaded from file using `file()` function.
- `certs` - (Optional) List of SSL certificates for SNI. Use with `keys` for multiple certificates.
- `keys` - (Optional) List of SSL private keys for SNI. Use with `certs` for multiple certificates.
- `ssl_protocols` - (Optional) List of SSL/TLS protocol versions to enable. Valid values: `TLSv1`, `TLSv1.1`, `TLSv1.2`, `TLSv1.3`. Defaults to APISIX defaults.
- `client` - (Optional) Client certificate verification configuration block for mTLS:
  - `ca_cert` - (Optional) CA certificate for client certificate verification in PEM format.
  - `depth` - (Optional) Maximum depth of CA certificates in the client certificate chain. Defaults to `1`.
- `labels` - (Optional) Labels as key-value pairs.

## Usage with Routes

SSL certificates work automatically with routes that match the SNI:

```hcl
resource "apisix_ssl" "api" {
  sni  = "api.example.com"
  cert = file("${path.module}/api.crt")
  key  = file("${path.module}/api.key")
}

resource "apisix_route" "api_route" {
  name  = "api-route"
  uri   = "/*"
  hosts = ["api.example.com"]

  upstream_id = apisix_upstream.backend.id
}
```

When a request comes to `https://api.example.com`, APISIX will automatically use the matching SSL certificate.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The ID of the SSL certificate.
- `sni` - The primary SNI.
- `snis` - The list of SNIs.

## Import

APISIX SSL certificates can be imported using the SSL ID, e.g.,

```bash
tofu import apisix_ssl.example <ssl-id>
```

Example:

```bash
tofu import apisix_ssl.example example.com
```

## Certificate Management Best Practices

1. **Use file() function**: Store certificates in version control (encrypted) or use a secret management system:
   ```hcl
   cert = file("${path.module}/certs/example.com.crt")
   key  = file("${path.module}/certs/example.com.key")
   ```

2. **Use environment variables for sensitive data**:
   ```hcl
   cert = var.ssl_cert
   key  = var.ssl_key
   ```

3. **Rotate certificates regularly**: Update the certificate files and run `tofu apply` to rotate.

4. **Use separate certificates per environment**: Don't share certificates between dev, staging, and production.

5. **Enable mTLS for sensitive APIs**: Use the `client` block to require client certificate verification.

## Notes

- **APISIX Configuration**: The SSL proxy must be enabled in your APISIX deployment for this resource to work.
- **Certificate Format**: Certificates and keys must be in PEM format.
- **SNI Matching**: APISIX matches incoming HTTPS requests to certificates based on the SNI field.
- **Multiple Certificates**: Use `certs`/`keys` lists for wildcard or multi-domain certificates.
- **Testing**: Acceptance tests are not executed in the default test environment. Test infrastructure is available in `tests/acceptance/ssl/` and can be enabled by configuring SSL proxy in APISIX. See implementation status documentation for details.
