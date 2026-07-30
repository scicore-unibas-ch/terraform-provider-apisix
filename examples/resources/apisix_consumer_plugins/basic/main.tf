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

# In a real deployment this consumer is created by whatever system owns user
# identities — a portal, an IdP sync, a provisioning script — along with its
# credential. It is declared here only so the example is runnable on its own.
#
# Do NOT manage the same consumer with apisix_consumer and
# apisix_consumer_plugins in your real configuration: apisix_consumer declares
# the complete plugins map, so the two would remove each other's plugins on
# every apply.
resource "apisix_consumer" "externally_owned" {
  id = "example-consumer-plugins-user"

  plugins = {
    "key-auth" = jsonencode({
      key = "example-credential-not-managed-below"
    })
  }

  # Required whenever both resources touch one consumer, as they do in this
  # self-contained example: apisix_consumer would otherwise see the plugin
  # attached below as drift and try to remove it on every plan.
  lifecycle {
    ignore_changes = [plugins]
  }
}

# Attach one plugin to that consumer without touching its credential.
resource "apisix_consumer_plugins" "basic" {
  consumer_id = apisix_consumer.externally_owned.id

  plugins = {
    "limit-count" = jsonencode({
      count         = 10000
      time_window   = 60
      rejected_code = 429
    })
  }
}
