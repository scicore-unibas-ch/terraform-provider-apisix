terraform {
  required_providers {
    apisix = {
      source = "scicore-unibas-ch/apisix"
    }
  }
}

variable "apisix_base_url" {
  type    = string
  default = "http://localhost:9180/apisix/admin"
}

variable "apisix_admin_key" {
  type      = string
  default   = "test123456789"
  sensitive = true
}

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
  timeout   = 30
}

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

resource "apisix_plugin_metadata" "http_logger" {
  id = "http-logger"
  metadata = jsonencode({
    log_format = {
      host         = "$host"
      "@timestamp" = "$time_iso8601"
      method       = "$request_method"
    }
  })
}

resource "apisix_plugin_metadata" "kafka_logger" {
  id = "kafka-logger"
  metadata = jsonencode({
    log_format = {
      host         = "$host"
      "@timestamp" = "$time_iso8601"
      upstream     = "$upstream_addr"
    }
  })
}
