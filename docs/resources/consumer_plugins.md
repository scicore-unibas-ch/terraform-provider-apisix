---
page_title: "apisix_consumer_plugins Resource - terraform-provider-apisix"
subcategory: ""
description: |-
  Manages individual plugins on an APISIX Consumer owned by another system, without taking ownership of the consumer or its credentials.
---

# apisix_consumer_plugins

Attaches plugins to an existing APISIX [Consumer](https://apisix.apache.org/docs/apisix/terminology/consumer/) **without owning it**.

Use this when something other than Terraform creates consumers — a self-service portal, an IdP sync job, a provisioning script — but you still want per-consumer plugin configuration (rate limits, quotas, restrictions) in version control.

Only the plugin keys you list are touched. Every other field of the consumer — `username`, `group_id`, `labels`, and any credential plugin such as `key-auth` or `jwt-auth` — is read and written back unchanged. Terraform never declares the credential, so an apply cannot invalidate it.

> Use [`apisix_consumer`](consumer.md) instead when Terraform owns the whole consumer, credentials included. Do not manage the same consumer with both resources: `apisix_consumer` declares the complete `plugins` map, so the two would each remove the other's plugins on every apply.

> **One `apisix_consumer_plugins` resource per consumer.** Writes are read-modify-write (see [Behaviour](#behaviour)), so two instances pointing at the same `consumer_id` race each other during an apply and one of the two writes is silently lost. Put every plugin you want on a consumer in a single resource.

## Example Usage

### Give one user a larger token quota than the route default

```hcl
resource "apisix_consumer_plugins" "power_user" {
  consumer_id = "alice"

  plugins = {
    "ai-rate-limiting" = jsonencode({
      limit_strategy = "total_tokens"
      rejected_code  = 429
      instances = [
        { name = "gpt-4o", limit = 10000000, time_window = 3600 },
      ]
    })
  }
}
```

Consumer-level plugin config outranks route-level config for that consumer (`Consumer > Route` in APISIX's precedence rules), so everyone else on the route keeps the default.

### Several plugins on one consumer

```hcl
resource "apisix_consumer_plugins" "restricted" {
  consumer_id = "contractor"

  plugins = {
    "limit-count" = jsonencode({
      count         = 500
      time_window   = 60
      rejected_code = 429
    })
    "ip-restriction" = jsonencode({
      whitelist = ["10.0.0.0/8"]
    })
  }
}
```

### Driven by a map of users

```hcl
variable "quota_exceptions" {
  type = map(object({
    count       = number
    time_window = number
  }))
  default = {}
}

resource "apisix_consumer_plugins" "quota" {
  for_each = var.quota_exceptions

  consumer_id = each.key

  plugins = {
    "limit-count" = jsonencode({
      count         = each.value.count
      time_window   = each.value.time_window
      rejected_code = 429
    })
  }
}
```

## Argument Reference

- `consumer_id` — (Required, ForceNew) Username of an **existing** consumer. Changing this forces replacement. The consumer must already exist; this resource never creates one.
- `plugins` — (Required) Map of plugin name to JSON-encoded configuration. These keys are merged into the consumer's existing plugins. Removing a key here detaches that plugin from the consumer on the next apply. JSON-equivalent values are suppressed, so server-side normalization does not produce drift.
- `timeouts` — (Optional) `create` / `read` / `update` / `delete` durations.

## Behaviour

**Writes are read-modify-write.** The APISIX Admin API does not support `PATCH` on `/consumers/{username}` (unlike routes, services and consumer groups), so every write is `GET` → merge the managed keys → `PUT` the whole object. Three consequences:

- **Not atomic.** If the owning system rewrites the same consumer between the read and the write, that write is lost. The window is milliseconds, and owning systems typically write the consumer only when creating a user — but do not run applies in a tight loop against a busy provisioner.
- **An external full rewrite drops the managed plugins.** A `PUT` from the owning system — re-provisioning a user, or any tool that writes the whole consumer — replaces the object, plugins included. The next `terraform apply` restores them; nothing else is lost, and plan output shows the resource as needing to be created again.

  You can design this away: keep the owning system's credentials in [Credential](https://apisix.apache.org/docs/apisix/terminology/credential/) objects (`/consumers/{username}/credentials/{id}`, APISIX 3.11+) rather than in the consumer's `plugins` map. Credential writes and consumer writes then touch different objects, so key rotation — the most frequent write in most provisioning systems — never disturbs the plugins managed here.
- **Delete detaches, it does not destroy.** `terraform destroy` removes the managed plugin keys and leaves the consumer, its credentials and its other plugins in place.

**Read only reflects managed keys.** Plugins added to the consumer by anyone else are ignored, not reported as drift. If every managed key disappears, the resource is removed from state.

## Import

The import ID is `<consumer>/<plugin>[,<plugin>...]`:

```bash
terraform import apisix_consumer_plugins.power_user 'alice/ai-rate-limiting'
terraform import apisix_consumer_plugins.restricted 'contractor/limit-count,ip-restriction'
```

The plugin names are **required**, and importing by bare consumer name is rejected. This is deliberate: importing everything on the consumer would pull `key-auth` into state, and the next apply would hand credential ownership to Terraform — the one outcome this resource exists to prevent.
