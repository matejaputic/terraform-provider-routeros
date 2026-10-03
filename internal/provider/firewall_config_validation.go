// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Configuration-only applicability: readback may retain inactive server fields
// after an action transition. Do not confuse those with configured mutation intent.
func (r *collectionResource) validateFirewallIntent(values map[string]attr.Value) error {
	if !r.orderedFirewall() {
		return nil
	}
	action, ok := values["action"].(types.String)
	if !ok || action.IsNull() || action.IsUnknown() {
		return nil
	}
	a := action.ValueString()
	if r.policy.Name == "ip_firewall_nat" {
		v := values["to_addresses"].(types.String)
		if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" && !oneOf(a, "src-nat", "dst-nat") {
			return fmt.Errorf("nonempty to_addresses is only implemented for src-nat/dst-nat")
		}
	}
	if r.policy.Name == "ip_firewall_mangle" {
		for _, field := range []struct{ name, action string }{{"new_connection_mark", "mark-connection"}, {"new_packet_mark", "mark-packet"}} {
			v := values[field.name].(types.String)
			if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" && a != field.action {
				return fmt.Errorf("configured %s requires action %s", field.name, field.action)
			}
		}
		p := values["passthrough"].(types.Bool)
		if !p.IsNull() && !p.IsUnknown() && !oneOf(a, "mark-connection", "mark-packet") {
			return fmt.Errorf("passthrough is only implemented for mark-connection/mark-packet")
		}
	}
	return nil
}
