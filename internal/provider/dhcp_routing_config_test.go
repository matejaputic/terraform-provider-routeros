// SPDX-License-Identifier: MPL-2.0
package provider

import "fmt"

var dhcpRoutingNames = []string{"ip_dhcp_server_network", "ip_dhcp_server", "ip_dhcp_server_lease", "ip_route"}

func dhcpRoutingConfig(provider, comment string) string {
	return networkingConfig(provider, comment) + fmt.Sprintf(`
resource "routeros_ip_address" "topology" {
 address = "192.0.2.1/24"
 interface = routeros_interface_bridge.test.name
 comment = %[1]q
}
resource "routeros_ip_dhcp_server_network" "test" {
 address = "192.0.2.0/24"
 gateway = "192.0.2.1"
 dns_server = ["192.0.2.53", "192.0.2.54"]
 dns_none = false
 netmask = 24
 domain = "example.test"
 comment = %[1]q
}
resource "routeros_ip_dhcp_server" "test" {
 name = "tf-coverage-dhcp"
 interface = routeros_interface_bridge.test.name
 address_pool = routeros_ip_pool.test.name
 disabled = true
 add_arp = false
 conflict_detection = true
 comment = %[1]q
 depends_on = [routeros_ip_address.topology, routeros_ip_dhcp_server_network.test]
}
resource "routeros_ip_dhcp_server_lease" "test" {
 address = "192.0.2.15"
 mac_address = "02:00:00:00:00:AA"
 server = routeros_ip_dhcp_server.test.name
 disabled = false
 block_access = false
 comment = %[1]q
}
resource "routeros_ip_route" "test" {
 dst_address = "198.51.100.0/24"
 gateway = "192.0.2.254"
 distance = 7
 scope = 30
 target_scope = 10
 routing_table = "main"
 disabled = true
 comment = %[1]q
 depends_on = [routeros_ip_address.topology]
}
`, comment)
}
