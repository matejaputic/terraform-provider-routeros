resource "routeros_ip_address" "example" {
  address   = "192.0.2.1/24"
  interface = "ether1"
  comment   = "Managed by Terraform"
  disabled  = false
}
