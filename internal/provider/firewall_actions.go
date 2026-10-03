// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func reviewedFirewallAction(family, action string) bool {
	switch family {
	case "ip_firewall_filter":
		return oneOf(action, "accept", "drop", "log", "passthrough", "return")
	case "ip_firewall_nat":
		return oneOf(action, "accept", "log", "passthrough", "return", "masquerade", "src-nat", "dst-nat")
	case "ip_firewall_mangle":
		return oneOf(action, "accept", "log", "passthrough", "return", "mark-connection", "mark-packet")
	case "ip_firewall_raw":
		return oneOf(action, "accept", "drop", "log", "notrack", "passthrough", "return")
	}
	return false
}
func (r *collectionResource) validateFirewallFields(values map[string]attr.Value, apply bool) error {
	if !r.orderedFirewall() {
		return nil
	}
	action := values["action"].(types.String)
	if action.IsNull() || action.IsUnknown() {
		return nil
	}
	required := ""
	if r.policy.Name == "ip_firewall_nat" && oneOf(action.ValueString(), "src-nat", "dst-nat") {
		required = "to_addresses"
	}
	if r.policy.Name == "ip_firewall_mangle" {
		switch action.ValueString() {
		case "mark-connection":
			required = "new_connection_mark"
		case "mark-packet":
			required = "new_packet_mark"
		}
	}
	if required != "" {
		v := values[required].(types.String)
		if v.IsNull() || !v.IsUnknown() && v.ValueString() == "" {
			return fmt.Errorf("%s requires nonempty %s", action.ValueString(), required)
		}
		if apply && v.IsUnknown() {
			return fmt.Errorf("action-specific field must be known")
		}
	}
	if v, ok := values["to_addresses"].(types.String); ok && !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		if _, e := parseRange(v.ValueString()); e != nil {
			return fmt.Errorf("to_addresses requires IPv4 address or ascending range (no CIDR/DNS/IPv6)")
		}
	}
	return nil
}
