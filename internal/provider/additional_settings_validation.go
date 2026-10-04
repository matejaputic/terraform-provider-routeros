// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"net/netip"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
)

// Explicit selected-field semantics; publication's device enums never grant
// permission for unsupported representations or nested settings.
func validateAdditionalSettings(policy catalog.Collection, values map[string]attr.Value) error {
	for _, f := range policy.Fields {
		v := values[f.Name]
		if v == nil || v.IsNull() || v.IsUnknown() {
			continue
		}
		if err := validateReviewedField(f, v); err != nil {
			return err
		}
		text, ok := v.(types.String)
		if !ok {
			continue
		}
		s := text.ValueString()
		switch {
		case policy.Name == "ip_dns" && f.Name == "servers":
			if s == "" {
				continue
			}
			seen := map[netip.Addr]bool{}
			for _, entry := range strings.Split(s, ",") {
				address, err := netip.ParseAddr(entry)
				if err != nil || address.Zone() != "" || address.IsUnspecified() || address.IsMulticast() || seen[address.Unmap()] {
					return fmt.Errorf("servers require unique unicast IP addresses")
				}
				seen[address.Unmap()] = true
			}
		case policy.Name == "snmp" && f.Name == "src_address":
			if s != "" {
				address, err := netip.ParseAddr(s)
				if err != nil || address.Zone() != "" || address.IsMulticast() {
					return fmt.Errorf("SNMP source must be an IP address")
				}
			}
		case policy.Name == "system_clock" && f.Name == "time_zone_name":
			if s != "manual" {
				if _, err := time.LoadLocation(s); err != nil {
					return fmt.Errorf("unsupported IANA time zone")
				}
			}
		case (policy.Name == "interface_l2tp_server" || policy.Name == "interface_sstp_server") && f.Name == "authentication":
			if s == "" {
				return fmt.Errorf("authentication methods cannot be empty")
			}
			for _, entry := range strings.Split(s, ",") {
				if !oneOf(entry, "pap", "chap", "mschap1", "mschap2") {
					return fmt.Errorf("unsupported authentication method")
				}
			}
		}
	}
	return nil
}
