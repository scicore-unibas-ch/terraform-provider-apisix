# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Plan-time validation of upstream mutual exclusions.** The cross-attribute rules APISIX enforces server-side now fail `validate`/`plan` instead of `apply`, on `apisix_upstream` and on the inline `upstream` block of `apisix_route` / `apisix_service`: `nodes` conflicts with `service_name` / `discovery_type` (and those two must be set together); `tls.client_cert` conflicts with `tls.client_cert_id` and must be paired with `tls.client_key`.

- **Go acceptance tests for every resource.** `TestAcc<Name>_stateStability` now exists for all 8 resources (previously only `consumer_group` and `plugin_metadata`), each covering create → in-place update (PUT full replace) → no-op re-plan → `ImportStateVerify`. Shared harness boilerplate moved to `internal/acctest`.
- **Unit tests for the HTTP client** (`internal/client`): httptest-backed coverage of the 5xx retry/backoff loop, retry exhaustion, no-retry on 4xx, the `ErrNotFound` sentinel, `force=true` delete, error-message fallbacks (`error_msg` → `message` → raw body), and context cancellation.
- **Unit tests for the shared upstream codec** (`internal/inlineupstream`): decode-defaults, legacy map-form nodes, health-check canonicalization, and a full wire round-trip fixture.
- Unit tests for `internal/timeoutshelper`, `internal/tfconv`, and `internal/pluginsmap`.

### Changed

- **Internal refactor — no schema or state changes.** Duplicated logic across resource packages was extracted into shared helpers:
  - `internal/pluginsmap` — the plugins-map codec (JSON validation with attribute-scoped diagnostics on the way in, canonical re-marshal on the way out), previously copy-pasted in six resources.
  - `internal/tfconv` — wire↔Terraform value conversions (`NullableString`, `StringPtr`, `CanonicalJSON`, …), previously re-declared per package.
  - `apisix_upstream` now composes `internal/inlineupstream` (schema, wire structs, codec) instead of maintaining a ~450-line parallel copy; the standalone resource and the inline `upstream` blocks of `apisix_route` / `apisix_service` are now guaranteed identical. Route's `timeout` block reuses the same codec.
  - Resource `Configure` boilerplate collapsed into `client.FromProviderData`.
- Attribute descriptions of the inline `upstream` block in `apisix_route` / `apisix_service` now match the standalone `apisix_upstream` resource verbatim (e.g. the `nodes` / `keepalive_pool` / `tls.client_cert` notes).
- Dead code removed: the unused `key` field of the Admin API response envelope, the never-read `commit` ldflag variable (also dropped from the goreleaser ldflags), and a no-op `UseStateForUnknown` plan modifier on `apisix_route.status` (redundant with its static default).

## [0.3.0] - 2026-05-19

### Added

- **New `apisix_plugin_metadata` resource.** Manages APISIX [Plugin Metadata](https://apisix.apache.org/docs/apisix/admin-api/#plugin-metadata) — per-plugin global configuration (log formats, exporter endpoints, default tracing parameters) applied to every instance of a given plugin. The resource follows the same pattern as `apisix_consumer_group`: `id` is the plugin name (`Required` + `RequiresReplace`), Update is `PUT` (full replace), and the JSON-encoded `metadata` body uses `jsonstring.SuppressEquivalent()` to absorb APISIX's server-side key reordering. The `Read` path strips `id` / `create_time` / `update_time` (which APISIX echoes into the response body) and recursively canonicalizes nested objects so `ImportStateVerify` round-trips cleanly. Includes Go acceptance test, bash acceptance fixtures for `syslog` / `http-logger` / `kafka-logger`, registry docs, and `examples/resources/apisix_plugin_metadata/{basic,advanced}/`.
- **Go acceptance test framework wired up.** Layered the Plugin Framework `resource.Test` harness on top of the existing docker-compose stack. The first test, `TestAccConsumerGroup_stateStability`, exercises three guarantees that the bash scripts cannot easily verify on the reference resource:
  - Create succeeds and the resource lands in state.
  - A re-plan against the same config produces an empty plan (state-stability — confirms that `Read` + the `jsonmap` plan modifier fully reconcile APISIX's server-side normalization).
  - Import by ID reconstructs the same state (`ImportStateVerify`).

  The test auto-skips unless `TF_ACC=1` is set, so `go test ./...` remains a fast unit-only run. `APISIX_BASE_URL` and `APISIX_ADMIN_KEY` configure the provider; the same env vars used by every other test path.
- **CI now runs `TF_ACC=1` Go acceptance tests** alongside the bash acceptance scripts in `acceptance-tests.yml`, against APISIX 3.14, 3.15, and 3.16.

### Changed

- **Minimum recommended APISIX version raised to 3.14.** APISIX 3.13.0 was dropped from the CI matrix (it remained on the 3.13.x branch but the project's reference target moved forward). Older versions may still work but are no longer verified.
- **Dependency bump:** `github.com/hashicorp/terraform-plugin-framework` `v1.16.1` → `v1.19.0` (required by the new `terraform-plugin-go v0.31.0` transitive dependency pulled in by `terraform-plugin-testing`).
- **Added `terraform-plugin-testing v1.16.0`** as a direct dependency for the new acceptance test framework.

### Fixed

- **Documentation clarifications** for attributes that are `Optional + Computed + Default` (so they appear in state after apply even when not set in HCL):
  - `apisix_route.priority`, `apisix_route.enable_websocket`, `apisix_route.status`
  - `apisix_service.enable_websocket`
  - `apisix_upstream.type`, `apisix_upstream.scheme`, `apisix_upstream.hash_on`, `apisix_upstream.pass_host`
- **All registry examples rewritten for the Plugin Framework schema.** Every `examples/resources/apisix_<name>/(basic|advanced)/main.tf` was carrying SDK v2 syntax that would not parse against v0.2.x. Corrections applied across all 14 files:
  - Replaced legacy URL-key attributes (`username`, `group_id`, `rule_id`, `config_id`, `name`-as-key) with the required `id = "..."` attribute on every resource.
  - Converted nested-object block syntax (`nodes { ... }`, `timeout { ... }`, inline `upstream { ... }`, `keepalive_pool { ... }`, `tls { ... }`) to attribute syntax (`nodes = [{ ... }]`, `timeout = { ... }`, etc.).
  - Fixed cross-resource references: `apisix_consumer_group.x.group_id` → `.id`, `apisix_plugin_config.x.config_id` → `.id`.
  - Fixed `apisix_service` examples that used a non-existent `api_key` provider attribute instead of `admin_key`.
  - Repaired `apisix_route` and `apisix_upstream` `basic`/`advanced` example files that started with markdown prose (broken HCL) and now contain valid standalone Terraform.
  - Bumped the version constraint in every example from `version = "0.1.0"` to `version = "~> 0.2"`.

## [0.2.0] - 2026-05-08

This is the **0.2.0** rewrite. The provider has been rebuilt on the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) (plugin protocol v6). It is a breaking change relative to 0.1.x; see the [Migration](#migrating-from-01x) section below.

### Changed (breaking)

- **Provider rewritten on the Terraform Plugin Framework.** The previous SDK v2 implementation has been removed. The provider now speaks plugin protocol v6 and is verified against APISIX 3.13, 3.14, 3.15, and 3.16 in CI.
- **`id` is now the explicit URL key on every resource** (`Required` + `RequiresReplace`). Previously the URL key was implicit (derived from `name`, `username`, `group_id`, `rule_id`, or `config_id`). All `.tf` files must add `id = "..."` to every resource.
- **Nested objects use attribute syntax instead of block syntax**: `nodes = [{...}]`, `timeout = {...}`, `keepalive_pool = {...}`, `tls = {...}`, inline `upstream = {...}`. The previous SDK v2 block syntax (`nodes { ... }`) is no longer accepted.
- **Cross-resource references that used the old per-resource keys must now use `.id`**: `apisix_consumer_group.x.group_id` → `apisix_consumer_group.x.id`, `apisix_plugin_config.x.config_id` → `apisix_plugin_config.x.id`.

### Added

- **Inline `upstream` block now exposes the full APISIX upstream surface** in `apisix_route` and `apisix_service` (scheme, timeouts, retries, hashing, keepalive, mTLS, service discovery, health checks). Previously inline upstreams supported only `type` and `nodes`.
- **JSON-equivalence plan modifiers** (`internal/planmodifier/jsonmap`, `internal/planmodifier/jsonstring`) for plugin maps and JSON-string fields (`vars`, `health_check`). APISIX's server-side normalization (key reordering, default injection) no longer produces perpetual diffs. Both modifiers carry their own unit tests covering key reordering, numeric type equivalence, partial equivalence, and null/invalid edge cases.
- **Per-resource `timeouts` blocks**. Every resource accepts:
  ```hcl
  timeouts {
    create = "30s"
    read   = "10s"
    update = "30s"
    delete = "10s"
  }
  ```
  The default for any unset operation is one minute, down from Terraform's framework default of 20 minutes.
- **Resilient HTTP client**: exponential-backoff retries on transient 5xx and network errors for idempotent verbs (GET, PUT, DELETE); configurable timeout, optional TLS skip-verify, structured `tflog` request/response tracing under `TF_LOG=TRACE` (DEBUG for retries only).
- **Plan-time validators** on `methods` (HTTP verbs), `port` (1-65535), `status` (0/1), `type` (load-balancing algorithm), `scheme` (upstream protocol), `hash_on`, `pass_host`.
- **`insecure` provider attribute** to opt into skipping TLS verification of the Admin API (off by default).
- **Registry manifest** (`terraform-registry-manifest.json`) declaring plugin protocol `6.0`, published as a release artifact via GoReleaser.
- **`go test ./...` and `go vet ./...` CI steps** ahead of acceptance tests so a Go-level regression fails fast without spinning up the full APISIX cluster.

### Fixed

- **Update is now `PUT` (full replace) instead of `PATCH` (merge).** Removing a field from `.tf` config now removes it server-side, rather than silently leaving stale values.
- **404 detection is a typed sentinel** (`client.ErrNotFound`) instead of `strings.Contains(err.Error(), "404")`.
- **`timeout` block no longer sends `0` for unset `connect`/`send`/`read`** fields, which previously overwrote APISIX's defaults.
- **Plugin JSON validation at plan time** — malformed plugin JSON now produces an attribute-scoped diagnostic instead of being silently dropped from the request.
- **Sensitive fields** (`tls.client_cert`, `tls.client_key`) marked correctly so they no longer leak into plan output.
- **`-X main.commit` ldflag** (already configured in `.goreleaser.yaml`) now has a target symbol declared in `main.go`.

### Removed

- **`apisix_ssl` resource** is not (yet) ported to the Plugin Framework rewrite. It will return in a future release. Existing SSL objects can still be managed via the APISIX Admin API directly while the resource is unavailable.
- **Stale repo-root status documents** (`IMPLEMENTATION_STATUS.md`, `VERIFICATION_REPORT.md`, `TEST_RESULTS.md`, `GITHUB_ACTIONS_SETUP.md`, `GITHUB_ACTIONS_SUMMARY.md`) — these described the prior SDK v2 implementation and were misleading on the registry repo page.

### Migrating from 0.1.x

There is **no automatic state migration** from the SDK v2 implementation; the rewrite intentionally treats this as a clean break. If you have existing state from a 0.1.x release of this provider:

1. Update your `.tf` files to the new schema:
   - Add `id = "..."` to every resource (use the value previously stored as `name` / `username` / `group_id` / `rule_id` / `config_id`).
   - Switch nested objects from block syntax to attribute syntax (`nodes { host = ... }` → `nodes = [{ host = ... }]`; same for `timeout`, `keepalive_pool`, `tls`, inline `upstream`).
   - Update cross-resource references: `apisix_consumer_group.x.group_id` → `apisix_consumer_group.x.id`, `apisix_plugin_config.x.config_id` → `apisix_plugin_config.x.id`.
2. Run `terraform state rm` for every existing resource managed by this provider.
3. Run `terraform import` for each resource using its APISIX object key.
4. Run `terraform plan` and confirm zero changes before applying.

## [0.1.x]

Earlier 0.1.x releases used the Terraform Plugin SDK v2. See git history for changes prior to the Plugin Framework rewrite.
