terraform {
  required_providers {
    routeros = { source = "matejaputic/routeros" }
  }
}

# Configure credentials through ROS_HOSTURL/ROS_USERNAME/ROS_PASSWORD.
provider "routeros" {}

resource "routeros_interface_bridge" "owned" {
  name = "tf-owned-batch-a"
}

resource "routeros_ipv6_pool" "owned" {
  name          = "tf-owned-v6-pool"
  prefix        = "2001:db8:107::/48"
  prefix_length = 64
}

resource "routeros_ipv6_address" "owned" {
  address   = "2001:db8:107::1/64"
  interface = routeros_interface_bridge.owned.name
  advertise = false
  disabled  = true
}

resource "routeros_routing_ospf_instance" "owned" {
  name      = "tf-owned-ospf"
  version   = 2
  router_id = "192.0.2.1"
  disabled  = true
}

resource "routeros_routing_ospf_area" "owned" {
  name     = "tf-owned-area"
  instance = routeros_routing_ospf_instance.owned.name
  area_id  = "0.0.0.107"
  disabled = true
}

resource "routeros_routing_ospf_interface_template" "owned" {
  area       = routeros_routing_ospf_area.owned.name
  interfaces = routeros_interface_bridge.owned.name
  cost       = 10
  disabled   = true
}

# Singleton destroy UNMANAGES only; it does not reset/delete the device note.
resource "routeros_system_note" "owned" {
  note          = "Terraform-owned test configuration"
  show_at_login = false
}

# Optional VETH configuration requires the enabled matching-version container
# package. See docs/development/batch-a.md; base-only devices fail closed.
