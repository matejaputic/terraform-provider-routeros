terraform {
  required_providers {
    routeros = {
      source = "matejaputic/routeros"
    }
  }
}

# REST connection credentials come from ROS_HOSTURL/ROS_USERNAME/ROS_PASSWORD.
provider "routeros" {}

# These definitions are not attached to a DHCP client or server.
resource "routeros_ip_dhcp_client_option" "vendor" {
  name  = "example-vendor-option"
  code  = 60
  value = "s'example-vendor'"
}

resource "routeros_ip_dhcp_server_option" "documentation" {
  name    = "example-documentation-option"
  code    = 77
  value   = "s'192.0.2.22'"
  force   = false
  comment = "Unattached documentation example"
}
