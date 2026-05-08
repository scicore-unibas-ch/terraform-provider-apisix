# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed (breaking)

- **Provider rewritten on the Terraform Plugin Framework.** The previous SDK v2 implementation has been removed in favor of a Plugin Framework provider speaking plugin protocol v6.
- **`id` is now the explicit URL key on every resource** (`Required` + `RequiresReplace`). Previously the URL key was implicit (derived from `name`, `username`, `group_id`, `rule_id`, or `config_id`). All `.tf` files must add `id = "..."` to every resource.
- **Nested objects use attribute syntax instead of block syntax**: `nodes = [{...}]`, `timeout = {...}`, `keepalive_pool = {...}`, `tls = {...}`, inline `upstream = {...}`. The previous SDK v2 block syntax (`nodes { ... }`) is no longer accepted.
- **Cross-resource references that used the old per-resource keys must now use `.id`**: `apisix_consumer_group.x.group_id` → `apisix_consumer_group.x.id`, `apisix_plugin_config.x.config_id` → `apisix_plugin_config.x.id`.

### Added

- **Inline `upstream` block now exposes the full APISIX upstream surface** in `apisix_route` and `apisix_service` (scheme, timeouts, retries, hashing, keepalive, mTLS, service discovery, health checks). Previously inline upstreams supported only `type` and `nodes`.
- **JSON-equivalence plan modifiers** for plugin maps and JSON-string fields (`vars`, `health_check`). APISIX's server-side normalization (key reordering, default injection) no longer produces perpetual diffs.
- **HTTP retries with exponential backoff** on idempotent verbs (GET, PUT, DELETE) for transient 5xx and network errors.
- **Validators** at plan time on `methods`, `port`, `status`, `type`, `scheme`, `hash_on`, `pass_host`.
- **`insecure` provider attribute** to opt into skipping TLS verification of the Admin API (off by default).
- **Registry manifest** (`terraform-registry-manifest.json`) declaring plugin protocol v6.

### Fixed

- **Update is now `PUT` (full replace) instead of `PATCH` (merge).** Removing a field from `.tf` config now removes it server-side, rather than silently leaving stale values.
- **404 detection is a typed sentinel** (`client.ErrNotFound`) instead of `strings.Contains(err.Error(), "404")`.
- **`timeout` block no longer sends `0` for unset `connect`/`send`/`read`** fields, which previously overwrote APISIX's defaults.
- **Plugin JSON validation at plan time** — malformed plugin JSON now produces an attribute-scoped diagnostic instead of being silently dropped from the request.
- **Sensitive fields** (`tls.client_cert`, `tls.client_key`) marked correctly so they no longer leak into plan output.

### Removed

- **`apisix_ssl` resource** is not (yet) ported to the Plugin Framework rewrite. It will return in a future release. Existing SSL objects can still be managed via the APISIX Admin API directly while the resource is unavailable.

### Migration notes

There is **no automatic state migration** from the SDK v2 implementation; the rewrite intentionally treats this as a clean break. If you have existing state from a 0.1.x release of this provider:

1. Apply the new schema to your `.tf` files (add `id`, switch to attribute syntax).
2. Run `terraform state rm` for every existing resource.
3. Run `terraform import` for each resource using its APISIX object key (the value previously stored as `name`/`username`/`group_id`/etc.).
4. Run `terraform plan` and confirm zero changes before applying.

## [0.1.x]

Earlier 0.1.x releases used the Terraform Plugin SDK v2. See git history for changes prior to the Plugin Framework rewrite.
