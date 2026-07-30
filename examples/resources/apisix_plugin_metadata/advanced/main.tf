terraform {
  required_providers {
    apisix = {
      source  = "scicore-unibas-ch/apisix"
      version = "~> 0.5"
    }
  }
}

provider "apisix" {
  base_url  = "http://localhost:9180/apisix/admin"
  admin_key = "test123456789"
}

# Configure http-logger to emit a structured JSON access log with request
# method, URI, response status, and upstream timing.
resource "apisix_plugin_metadata" "http_logger" {
  id = "http-logger"
  metadata = jsonencode({
    log_format = {
      host          = "$host"
      "@timestamp"  = "$time_iso8601"
      method        = "$request_method"
      uri           = "$request_uri"
      status        = "$status"
      remote_addr   = "$remote_addr"
      upstream      = "$upstream_addr"
      upstream_time = "$upstream_response_time"
      user_agent    = "$http_user_agent"
    }
  })
}

# kafka-logger sharing the same log shape — useful when you ship the
# same access log to two destinations.
resource "apisix_plugin_metadata" "kafka_logger" {
  id = "kafka-logger"
  metadata = jsonencode({
    log_format = {
      host          = "$host"
      "@timestamp"  = "$time_iso8601"
      method        = "$request_method"
      uri           = "$request_uri"
      status        = "$status"
      remote_addr   = "$remote_addr"
      upstream      = "$upstream_addr"
      upstream_time = "$upstream_response_time"
    }
  })
}

# Reference a route that uses these plugins. The metadata above applies
# to every invocation of the syslog / http-logger / kafka-logger plugin
# anywhere in APISIX; the route only needs to enable the plugin.
resource "apisix_route" "logged" {
  id   = "logged-route"
  uri  = "/logged/*"
  name = "Logged route"

  upstream = {
    type  = "roundrobin"
    nodes = [{ host = "127.0.0.1", port = 8080, weight = 100 }]
  }

  plugins = {
    "http-logger" = jsonencode({
      uri = "https://example.com/logs"
    })
    "kafka-logger" = jsonencode({
      brokers       = [{ host = "kafka.internal", port = 9092 }]
      kafka_topic   = "apisix-access"
      key           = "access-log"
      batch_max_size = 1000
    })
  }
}
