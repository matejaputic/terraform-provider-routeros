// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"strings"
)

func firewallFamilyConfig(family, provider, comment, before string) string {
	cfg := firewallConfig(provider, comment, before)
	cfg = strings.ReplaceAll(cfg, "ip_firewall_filter", "ip_firewall_"+family)
	cfg = strings.ReplaceAll(cfg, "tf-coverage-filter", "tf-coverage-"+family)
	cfg = strings.ReplaceAll(cfg, `action = "drop"`, `action = "return"`)
	switch family {
	case "nat":
		addr := "192.0.2.128"
		if comment != "first" {
			addr = "192.0.2.129"
		}
		cfg = strings.ReplaceAll(cfg, `action = "accept"`, fmt.Sprintf("action = \"src-nat\"\n to_addresses = %q", addr))
	case "mangle":
		cfg = strings.ReplaceAll(cfg, `action = "accept"`, fmt.Sprintf("action = \"mark-connection\"\n new_connection_mark = %q\n passthrough = false", "tf-mark-"+comment))
	case "raw":
		action := "notrack"
		if comment != "first" {
			action = "drop"
		}
		cfg = strings.ReplaceAll(cfg, `action = "accept"`, fmt.Sprintf("action = %q", action))
	}
	return cfg
}
