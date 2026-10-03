// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"
)

func firewallAddressBounds(text string) (ipRange, error) {
	if strings.Contains(text, "/") {
		p, e := netip.ParsePrefix(text)
		if e != nil || !p.Addr().Is4() || p != p.Masked() {
			return ipRange{}, fmt.Errorf("firewall address requires canonical IPv4 CIDR")
		}
		a := p.Addr().As4()
		end := binary.BigEndian.Uint32(a[:]) | (^uint32(0) >> p.Bits())
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], end)
		return ipRange{p.Addr(), netip.AddrFrom4(b)}, nil
	}
	r, e := parseRange(text)
	if e != nil {
		return ipRange{}, fmt.Errorf("firewall address requires IPv4 address, canonical CIDR or ascending range; DNS/IPv6/expiry are not implemented")
	}
	return r, nil
}
