// SPDX-License-Identifier: MPL-2.0
package provider

import "fmt"

func firewallConfig(provider, comment, before string) string {
	return provider + fmt.Sprintf(`
resource "routeros_ip_firewall_addr_list" "test" {
 list = "tf-coverage-firewall"
 address = "192.0.2.0-192.0.2.255"
 comment = %[1]q
 disabled = false
}
resource "routeros_ip_firewall_filter" "tail" {
 chain = "tf-coverage-filter"
 action = "drop"
 disabled = true
 log = false
 comment = "tf-coverage-tail"
}
resource "routeros_ip_firewall_filter" "head" {
 chain = "tf-coverage-filter"
 action = "accept"
 src_address_list = routeros_ip_firewall_addr_list.test.list
 disabled = true
 log = false
 comment = %[1]q
 place_before = %[2]s
}
`, comment, before)
}
