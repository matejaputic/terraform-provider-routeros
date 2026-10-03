terraform {
  required_providers {
    routeros = {
      source = "matejaputic/routeros"
    }
  }
}

# Set ROS_HOSTURL, ROS_USERNAME and ROS_PASSWORD; do not commit credentials.
provider "routeros" {}

# Choose an UNUSED interface. Do not attach the management interface.
variable "unused_interface" {
  type        = string
  description = "An unused Ethernet interface that can safely join this new bridge."
}

resource "routeros_interface_bridge" "example" {
  name           = "tf-example-bridge"
  protocol_mode  = "rstp"
  vlan_filtering = false
  comment        = "Terraform managed"
}

resource "routeros_interface_bridge_port" "example" {
  bridge    = routeros_interface_bridge.example.name
  interface = var.unused_interface
  comment   = "Terraform managed"
}

resource "routeros_interface_vlan" "example" {
  name      = "tf-example-vlan"
  interface = routeros_interface_bridge.example.name
  vlan_id   = 123
}

resource "routeros_interface_list" "example" {
  name    = "tf-example-list"
  comment = "Terraform managed"
}

resource "routeros_interface_list_member" "example" {
  list      = routeros_interface_list.example.name
  interface = routeros_interface_vlan.example.name
}

resource "routeros_ip_pool" "example" {
  name   = "tf-example-pool"
  ranges = ["192.0.2.10-192.0.2.20", "192.0.2.30-192.0.2.40"]
}
