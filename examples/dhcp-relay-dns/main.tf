# Disposable-router example. The bridge has no ports, both objects are disabled.
terraform {
  required_providers {
    routeros = {
      source = "matejaputic/routeros"
    }
  }
}

provider "routeros" {} # Use ROS_HOSTURL, ROS_USERNAME and ROS_PASSWORD.

resource "routeros_interface_bridge" "isolated" {
  name          = "tf-relay-isolated"
  protocol_mode = "none"
}

resource "routeros_ip_dhcp_relay" "isolated" {
  name           = "tf-relay-example"
  interface      = routeros_interface_bridge.isolated.name
  dhcp_server    = "192.0.2.1,192.0.2.2"
  disabled       = true
  local_address  = "0.0.0.0"
  add_relay_info = false
}

resource "routeros_ip_dns_record" "isolated" {
  name            = "tf-example.invalid"
  type            = "A"
  address         = "192.0.2.10"
  comment         = "disabled example"
  disabled        = true
  match_subdomain = false
}
