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

# Per-plugin global metadata. The `id` is the plugin name and the
# `metadata` body shape is plugin-specific — consult the docs for the
# plugin you are configuring.
resource "apisix_plugin_metadata" "syslog" {
  id = "syslog"
  metadata = jsonencode({
    log_format = {
      host         = "$host"
      "@timestamp" = "$time_iso8601"
      client_ip    = "$remote_addr"
    }
  })
}
