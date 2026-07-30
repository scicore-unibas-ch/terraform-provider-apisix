terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "~> 0.3"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

# --- Fixtures ----------------------------------------------------------------
# Stand-ins for consumers created by the system that owns user identities.
# See the basic example for why ignore_changes is needed when one config both
# creates a consumer and decorates it.

resource "apisix_consumer" "externally_owned" {
  for_each = toset(["example-power-user", "example-contractor"])

  id = each.key

  plugins = {
    "key-auth" = jsonencode({ key = "credential-for-${each.key}" })
  }

  lifecycle {
    ignore_changes = [plugins]
  }
}

# --- Per-user exceptions -----------------------------------------------------
# The pattern this resource is built for: a default limit lives on the route,
# and named individuals get their own. Consumer-level plugin config outranks
# route-level for that consumer, so everyone else keeps the default.
#
# IMPORTANT: exactly ONE apisix_consumer_plugins resource per consumer. Writes
# are read-modify-write (the Admin API has no PATCH for consumers), so two
# instances pointing at the same consumer race each other and one write is
# lost. Put every plugin for a consumer in the same resource, as the
# contractor entry below does.

variable "consumer_plugins" {
  description = "Plugins to attach per consumer, keyed by consumer username."
  type        = map(map(string))

  default = {
    example-power-user = {
      # A quota well above whatever the route grants by default.
      "limit-count" = <<-JSON
        { "count": 100000, "time_window": 60, "rejected_code": 429 }
      JSON
    }

    example-contractor = {
      # Tighter limits AND a network restriction — one resource, three plugins.
      "limit-count" = <<-JSON
        { "count": 200, "time_window": 60, "rejected_code": 429 }
      JSON
      "limit-req" = <<-JSON
        { "rate": 5, "burst": 10, "rejected_code": 429,
          "key_type": "var", "key": "consumer_name" }
      JSON
      "ip-restriction" = <<-JSON
        { "whitelist": ["10.0.0.0/8", "192.168.0.0/16"] }
      JSON
    }
  }
}

resource "apisix_consumer_plugins" "exceptions" {
  for_each = var.consumer_plugins

  consumer_id = apisix_consumer.externally_owned[each.key].id
  plugins     = each.value
}

output "managed_consumers" {
  description = "Consumers decorated here. Their credentials are never declared by this resource, so applies cannot invalidate them."
  value       = keys(var.consumer_plugins)
}
