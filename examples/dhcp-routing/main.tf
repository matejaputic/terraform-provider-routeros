terraform {
  required_providers {
    routeros = { source = "matejaputic/routeros" }
  }
}

# Configure credentials through ROS_* environment variables.
provider "routeros" {}

# A new disconnected bridge: never use a management interface for experiments.
resource "routeros_interface_bridge" "isolated" {
  name = "tf-example-dhcp"
}
resource "routeros_ip_address" "gateway" {
  address   = "192.0.2.1/24"
  interface = routeros_interface_bridge.isolated.name
}
resource "routeros_ip_pool" "clients" {
  name   = "tf-example-dhcp-pool"
  ranges = ["192.0.2.10-192.0.2.20"]
}
resource "routeros_ip_dhcp_server_network" "clients" {
  address    = "192.0.2.0/24"
  gateway    = "192.0.2.1"
  dns_server = ["192.0.2.53"]
  dns_none   = false
}
resource "routeros_ip_dhcp_server" "clients" {
  name         = "tf-example-dhcp-server"
  interface    = routeros_interface_bridge.isolated.name
  address_pool = routeros_ip_pool.clients.name
  disabled     = true # Example config only; enabling requires an appropriate isolated L2 network.
  depends_on   = [routeros_ip_address.gateway, routeros_ip_dhcp_server_network.clients]
}
resource "routeros_ip_dhcp_server_lease" "example" {
  address     = "192.0.2.15"
  mac_address = "02:00:00:00:00:AA"
  server      = routeros_ip_dhcp_server.clients.name
}
resource "routeros_ip_route" "example" {
  dst_address = "198.51.100.0/24" # Never implicitly creates a default route.
  gateway     = "192.0.2.254"
  distance    = 7
  disabled    = true
  depends_on  = [routeros_ip_address.gateway]
}
