# CLAUDE.md

This file orients Claude Code agents working in this repo. Read it before making non-trivial changes.

## Project

OpenTofu/Terraform provider for the Apache APISIX Admin API, built on **terraform-plugin-framework** (plugin protocol v6). Provider address: `registry.opentofu.org/scicore-unibas-ch/apisix`.

The provider was rewritten from SDK v2 to the Plugin Framework in v0.2.0 — clean break, no state migration (zero existing users; `dev_overrides`-only at the time). Design rationale lives in the `project-rewrite-plan` memory entry.

## Repository layout

- `internal/client/` — hardened HTTP client (PUT for Update, `ErrNotFound` sentinel, idempotent retries with backoff, optional TLS skip-verify). Also exports `FromProviderData` — the shared resource `Configure` helper.
- `internal/provider/` — provider entrypoint and resource registration.
- `internal/planmodifier/jsonmap/` — Map plan modifier for plugin maps (e.g. `consumer_group.plugins`).
- `internal/planmodifier/jsonstring/` — String plan modifier for single JSON-string fields (e.g. `route.vars`, `plugin_metadata.metadata`). **Do not create a new String JSON modifier — this one already exists.**
- `internal/inlineupstream/` — the single source of truth for the APISIX upstream object: schema attrs, wire structs, and codec. Used by the inline `upstream` blocks of route/service **and** by the standalone `apisix_upstream` resource (whose model embeds `inlineupstream.Fields`). Change upstream attributes here only.
- `internal/pluginsmap/` — shared codec for `plugins` map attributes (`Build` validates each value as JSON with attribute-scoped diagnostics; `Decode` canonicalizes). **Do not hand-roll plugin map conversion in a resource.**
- `internal/tfconv/` — wire↔Terraform value conversions (`NullableString`, `StringPtr`, `CanonicalJSON`, …). Reach for these before writing a local helper.
- `internal/acctest/` — shared Go acceptance-test harness (provider factories + `PreCheck`).
- `internal/resource/<name>/` — one package per resource. `consumergroup` is the reference for the standard shape; `pluginmetadata` is the reference for resources whose body is a single JSON blob.
- `internal/timeoutshelper/` — `timeouts {}` block glue.
- `tests/acceptance/<name>/{main.tf,test.sh}` — bash acceptance harness, run by `make test-acceptance`.
- `internal/resource/<name>/resource_test.go` — Go `resource.Test` acceptance tests, gated on `TF_ACC=1`.
- `docs/resources/<name>.md` — OpenTofu Registry docs, hand-written (not `tfplugindocs`-generated).
- `examples/resources/apisix_<name>/{basic,advanced}/main.tf` — registry example fixtures.
- `.github/workflows/acceptance-tests.yml` — per-resource bash test step, runs against APISIX 3.14 / 3.15 / 3.16.

## Resource conventions (use `consumergroup` and `pluginmetadata` as references)

- `id` is `Required` + `RequiresReplace`. Never reuse `name` as the URL key.
- Update calls `client.Put` (full replace). Never PATCH.
- On Create and Update, set state = plan verbatim (Framework strict mode). Let Read reconcile drift.
- API body uses pointer fields (`*string`) so omitted vs. empty is distinguishable. Convert with `tfconv.StringPtr` / `Int64Ptr` / `BoolPtr`; decode with `tfconv.NullableString` / `OptInt64` / `StringOrDefault` etc.
- `Configure` is one line: `r.client = client.FromProviderData(req.ProviderData, &resp.Diagnostics)`.
- Plugins maps: `pluginsmap.Build` in `buildBody` (validates each value as JSON with an attribute-scoped diagnostic — never silently drop) and `pluginsmap.Decode` in `decodeInto` (canonicalizes; `emptyAsNull=true` for Optional plugins, `false` for Required). Single-JSON-blob attrs (`metadata`, `vars`, `health_check`): validate in `buildBody`, decode via `tfconv.CanonicalJSON` — canonical key order is required for `ImportStateVerify` text equality.
- Plugin maps → `jsonmap.SuppressEquivalent()`. Single JSON-string attrs → `jsonstring.SuppressEquivalent()`.
- Cross-attribute rules APISIX enforces server-side should fail at plan time: `resourcevalidator.Conflicting`/`RequiredTogether` in `ConfigValidators`, or a custom attribute validator for conditional requirements (see `rewriteRequiresUpstreamHost` in inlineupstream, which uses `req.Path.ParentPath()` so it works at root and nested levels).
- APISIX echoes synthetic fields into GET response bodies (`id`, `create_time`, `update_time`). Strip them in `decodeInto`.

## Adding a resource: full checklist

1. `internal/resource/<name>/resource.go` — Plugin Framework resource (Create/Read/Update/Delete/ImportState).
2. `internal/resource/<name>/resource_test.go` — `TestAcc<Name>_stateStability` covering apply → in-place update → no-op re-plan → `ImportStateVerify`. Use the `internal/acctest` harness (`acctest.PreCheck`, `acctest.ProtoV6ProviderFactories`). Add an `ExpectError` test for any ConfigValidators.
3. `tests/acceptance/<name>/main.tf` + `test.sh` (chmod +x) — multi-fixture bash harness: create, no-op replan, destroy, import round-trip.
4. `docs/resources/<name>.md` — frontmatter + Example Usage + Argument Reference + Import section.
5. `examples/resources/apisix_<name>/{basic,advanced}/main.tf` — runnable examples. Apply + no-op replan + destroy against the live APISIX docker stack **before committing**.
6. Register `NewResource` in `internal/provider/provider.go` `Resources()`.
7. Add a step to `.github/workflows/acceptance-tests.yml`.
8. CHANGELOG entry under `[Unreleased] > ### Added`.
9. README resources table + `docs/index.md` "Supported Resources" list.

## Resource status

| Resource | Status |
| --- | --- |
| apisix_consumer | done |
| apisix_consumer_group | done — reference impl |
| apisix_global_rule | done |
| apisix_plugin_config | done |
| apisix_plugin_metadata | done in v0.3.0 — second reference (single-JSON-blob body shape) |
| apisix_route | done |
| apisix_service | done |
| apisix_upstream | done |
| **apisix_ssl** | **TODO — required for v1.0** (was item 8 of the original locked plan). Hardest of the in-plan resources because APISIX returns the private key encrypted; Read must preserve the plan-side value rather than reconciling from API. `Sensitive: true` on `private_key`, optional SNI parsing from PEM. README currently tells users this resource is missing. |
| apisix_stream_route | not started — easy-medium. Flat 5-field schema (`upstream_id`, `remote_addr`, `server_addr`, `server_port`, `sni`). Server-assigned ID (Computed) breaks the `id=Required` convention — decide whether to break the pattern or force users to supply IDs. Requires APISIX `stream_proxy.tcp` enabled in `tests/docker-compose.yml` for acceptance. |
| apisix_secret | not started — hardest. Three sub-variants (vault/aws/gcp), each ~70-100 lines of nested schema. URL is `/secrets/{manager}/{id}` — two-part path; works with the current client by passing `kind = "secrets/vault"`. Custom `ImportState` (format `<manager>/<id>`), `ConfigValidators` with `ExactlyOneOf`, sensitive fields throughout. |

## Reference repo (do not copy code from it)

`terraform-provider-apisix-rework-space-com/` is a downloaded copy of a separate APISIX provider (`holubovskyi/apisix-client-go`-based). It implements `plugin_metadata`, `secret`, `ssl_certificate`, and `stream_route`. Useful for cross-referencing API request/response shapes, but its idioms (no plan modifiers, untyped maps, partial state preservation, SDK-v2-style debug logging) do not match this repo's conventions. Use for API discovery, not copy-paste.

## Testing

- `make test` — unit tests (no APISIX needed).
- `make test-env-up` / `make test-env-down` — APISIX docker-compose stack.
- `make test-acceptance` — full bash + Go acceptance suite against the stack.
- Go-only acceptance: `TF_ACC=1 APISIX_BASE_URL=http://localhost:9180/apisix/admin APISIX_ADMIN_KEY=test123456789 go test ./... -run '^TestAcc' -v`.
- **Rebuild the binary** (`go build -o terraform-provider-apisix .`) after every provider-code change before bash tests — `dev_overrides` loads the binary, not source.
- CI matrix: APISIX 3.14 / 3.15 / 3.16.

## Release process

Tag-driven via `.github/workflows/release.yml` (goreleaser + GPG). To cut release `vX.Y.Z`:

1. Rename `[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD` in `CHANGELOG.md` and open a fresh `[Unreleased]` heading above it.
2. For a minor bump, bump `version = "~> X.Y"` in `examples/resources/*/*/main.tf`, `README.md` quick-install, and `docs/index.md` provider example.
3. `git tag -s vX.Y.Z -m "vX.Y.Z"` and `git push origin vX.Y.Z`.
