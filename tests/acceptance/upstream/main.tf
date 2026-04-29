terraform {
  required_providers {
    apisix = {
      source = "scicore/apisix"
    }
  }
}

provider "apisix" {
  base_url  = var.apisix_base_url
  admin_key = var.apisix_admin_key
  timeout   = 30
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

# Test 1: Basic upstream with single node
resource "apisix_upstream" "basic" {
  name = "test-upstream-basic"
  type = "roundrobin"
  desc = "Basic test upstream with single node"

  nodes {
    host   = "127.0.0.1"
    port   = 8080
    weight = 100
  }

  scheme = "http"

  labels = {
    env  = "test"
    team = "platform"
  }
}

# Test 2: Medium complexity upstream with multiple nodes and timeouts
resource "apisix_upstream" "medium" {
  name        = "test-upstream-medium"
  type        = "roundrobin"
  desc        = "Medium complexity upstream with multiple nodes and timeouts"
  scheme      = "http"
  retries     = 3
  retry_timeout = 10

  nodes {
    host     = "127.0.0.1"
    port     = 8080
    weight   = 100
    priority = 0
  }

  nodes {
    host     = "127.0.0.1"
    port     = 8081
    weight   = 50
    priority = 1
  }

  timeout {
    connect = 5
    send    = 10
    read    = 15
  }

  labels = {
    env      = "test"
    team     = "platform"
    complexity = "medium"
  }
}

# Test 3: Complex upstream with all supported fields
resource "apisix_upstream" "complex" {
  name          = "test-upstream-complex"
  type          = "chash"
  desc          = "Complex upstream with all supported fields"
  scheme        = "http"
  hash_on       = "vars"
  key           = "remote_addr"
  pass_host     = "pass"
  retries       = 2
  retry_timeout = 5

  nodes {
    host     = "127.0.0.1"
    port     = 9080
    weight   = 100
    priority = 0
    metadata = {
      version = "v1"
      zone    = "us-east-1"
    }
  }

  nodes {
    host     = "127.0.0.1"
    port     = 9081
    weight   = 50
    priority = 1
  }

  timeout {
    connect = 3
    send    = 5
    read    = 10
  }

  health_check = jsonencode({
    active = {
      http_path = "/health"
      interval  = 5
      timeout   = 3
      concurrency = 10
      type      = "http"
      healthy = {
        interval  = 3
        successes = 2
        http_statuses = [200, 302]
      }
      unhealthy = {
        interval      = 3
        http_failures = 3
        tcp_failures  = 2
        timeouts      = 3
        http_statuses = [429, 500, 502, 503, 504]
      }
    }
    passive = {
      type = "http"
      healthy = {
        http_statuses = [200, 201, 202, 301, 302]
        successes     = 5
      }
      unhealthy = {
        http_failures = 5
        tcp_failures  = 2
        timeouts      = 7
        http_statuses = [429, 500, 503]
      }
    }
  })

  keepalive_pool {
    size         = 320
    idle_timeout = 60
    requests     = 1000
  }

  labels = {
    env        = "test"
    team       = "platform"
    complexity = "high"
    purpose    = "acceptance-test"
  }
}
