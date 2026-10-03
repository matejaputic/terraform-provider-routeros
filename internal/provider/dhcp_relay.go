// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"net/netip"
	"strings"
)

// RouterOS dhcp-server is a comma-separated IPv4 list, not a Terraform set.
// Reject whitespace, duplicates and noncanonical spelling rather than silently
// changing configured values during readback. 0.0.0.0 is the documented local
// address auto-selection sentinel; server destinations must be unicast.
func validateDHCPRelayString(field, text string) error {
	switch field {
	case "dhcp_server":
		seen := map[netip.Addr]bool{}
		for _, entry := range strings.Split(text, ",") {
			ip, err := netip.ParseAddr(entry)
			if err != nil || !ip.Is4() || ip.String() != entry || ip.IsUnspecified() || ip.IsMulticast() || entry == "255.255.255.255" || seen[ip] {
				return fmt.Errorf("dhcp_server requires unique canonical unicast IPv4 addresses separated by commas")
			}
			seen[ip] = true
		}
	case "local_address":
		ip, err := netip.ParseAddr(text)
		if err != nil || !ip.Is4() || ip.String() != text || ip.IsMulticast() || text == "255.255.255.255" {
			return fmt.Errorf("local_address requires a canonical IPv4 address")
		}
	}
	return nil
}
