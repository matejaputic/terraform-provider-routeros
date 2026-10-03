terraform {
  required_providers {
    routeros = { source = "matejaputic/routeros" }
  }
}
# Use ROS_* environment variables for credentials.
provider "routeros" {}

resource "routeros_ip_firewall_addr_list" "example" {
  list    = "tf-example-doc-addresses"
  address = "192.0.2.0/24"
}
# Unreferenced custom chain + disabled rules: this example cannot secure traffic
# until explicitly integrated. Never experiment on management-access rules.
resource "routeros_ip_firewall_filter" "tail" {
  chain    = "tf-example-isolated"
  action   = "drop"
  disabled = true
}
resource "routeros_ip_firewall_filter" "head" {
  chain            = "tf-example-isolated"
  action           = "accept"
  src_address_list = routeros_ip_firewall_addr_list.example.list
  disabled         = true
  place_before     = routeros_ip_firewall_filter.tail.id
  # Explicit "" moves to chain end; omitted leaves placement unmanaged.
}
