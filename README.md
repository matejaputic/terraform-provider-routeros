# Terraform Provider for RouterOS (Automatically generated)

This provider combines schemas and models automatically generated from versioned RouterOS OpenAPI descriptions with reviewed lifecycle code for safe REST-based resource management. Its continuous-maintenance pipeline detects schema drift and validates generated updates through deterministic generation, contract checks, and lifecycle tests before promotion.

## Usage Example

```hcl
terraform {
  required_providers {
    routeros = {
      source = "matejaputic/routeros"
    }
  }
}

# Configure ROS_HOSTURL, ROS_USERNAME, and ROS_PASSWORD in your environment.
# Registry publication is pending; use a local development installation for now.
provider "routeros" {}

variable "lan_interface" {
  type        = string
  description = "An unused Ethernet interface to attach; never select the management interface."
}

resource "routeros_interface_bridge" "lan" {
  name          = "terraform-lan"
  protocol_mode = "rstp"
}

resource "routeros_interface_bridge_port" "lan" {
  bridge    = routeros_interface_bridge.lan.name
  interface = var.lan_interface
}

# Choose a subnet that does not overlap your existing networks.
resource "routeros_ip_address" "gateway" {
  address   = "192.168.50.1/24"
  interface = routeros_interface_bridge.lan.name
}

# This pool can be referenced by a DHCP server configured separately.
resource "routeros_ip_pool" "lan" {
  name   = "terraform-lan-pool"
  ranges = ["192.168.50.100-192.168.50.200"]
}
```
