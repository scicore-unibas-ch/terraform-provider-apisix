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

# NOTE: the consumers these fixtures decorate are NOT created here. test.sh
# creates them over the Admin API first, standing in for the external system
# (a portal, an IdP) that owns consumers and their credentials. Creating them
# with apisix_consumer instead would make Terraform own the whole object, and
# the two resources would fight over the same plugins map — which is exactly
# the situation apisix_consumer_plugins exists to avoid.

# One managed plugin on a consumer that also has a key-auth credential.
resource "apisix_consumer_plugins" "basic" {
  consumer_id = "test-consumer-plugins-basic"

  plugins = {
    "limit-count" = jsonencode({
      count         = 10000
      time_window   = 60
      rejected_code = 429
    })
  }
}

# Two managed plugins at once, to prove the merge handles multiple keys.
resource "apisix_consumer_plugins" "multi" {
  consumer_id = "test-consumer-plugins-multi"

  plugins = {
    "limit-count" = jsonencode({
      count         = 500
      time_window   = 60
      rejected_code = 429
    })
    "limit-req" = jsonencode({
      rate          = 10
      burst         = 20
      rejected_code = 429
      key_type      = "var"
      key           = "consumer_name"
    })
  }
}
