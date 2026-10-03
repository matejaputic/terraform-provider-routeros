terraform {
  required_providers {
    routeros = { source = "matejaputic/routeros" }
  }
}
# ROS_* environment credentials. Disabled, unreferenced chains only.
provider "routeros" {}

resource "routeros_ip_firewall_nat" "example" {
  chain        = "tf-example-nat"
  action       = "src-nat"
  to_addresses = "192.0.2.128"
  disabled     = true
}
resource "routeros_ip_firewall_mangle" "example" {
  chain               = "tf-example-mangle"
  action              = "mark-connection"
  new_connection_mark = "tf-example-mark"
  passthrough         = false
  disabled            = true
}
resource "routeros_ip_firewall_raw" "tail" {
  chain    = "tf-example-raw"
  action   = "return"
  disabled = true
}
resource "routeros_ip_firewall_raw" "head" {
  chain        = "tf-example-raw"
  action       = "notrack"
  disabled     = true
  place_before = routeros_ip_firewall_raw.tail.id
}
