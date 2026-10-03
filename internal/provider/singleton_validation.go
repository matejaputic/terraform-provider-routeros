// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"net/netip"
	"strings"
)

func singletonAddressList(resource, field string) bool {
	return resource == "system_ntp_client" && field == "servers" || resource == "system_ntp_server" && field == "broadcast_addresses"
}
func validateSingletonString(resource, field, text string) error {
	if singletonAddressList(resource, field) {
		if text == "" {
			return nil
		} // Explicit empty list clears; null omits.
		seen := map[netip.Addr]bool{}
		for _, entry := range strings.Split(text, ",") {
			ip, err := netip.ParseAddr(entry)
			if err != nil || ip.Zone() != "" || ip.String() != entry || ip.Is4In6() || seen[ip] {
				return fmt.Errorf("%s requires unique canonical IP addresses separated by commas", field)
			}
			if resource == "system_ntp_server" && !ip.Is4() {
				return fmt.Errorf("broadcast_addresses requires IPv4 addresses")
			}
			seen[ip] = true
		}
	}
	switch field {
	case "name", "allowed_interface_list", "discover_interface_list", "vrf":
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("%s must be nonempty", field)
		}
	}
	return nil
}
func equivalentAddressList(a, b string) bool {
	left := map[string]int{}
	for _, entry := range strings.Split(a, ",") {
		left[entry]++
	}
	for _, entry := range strings.Split(b, ",") {
		left[entry]--
		if left[entry] < 0 {
			return false
		}
	}
	for _, count := range left {
		if count != 0 {
			return false
		}
	}
	return true
}
