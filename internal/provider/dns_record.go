// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Explicit useful subset: named A/AAAA address records. Other DNS actions and
// regex matching need their own reviewed payload/readback contracts.
func validateDNSAddressRecord(values map[string]attr.Value) error {
	known := func(field string) (string, bool) {
		v, ok := values[field].(types.String)
		return v.ValueString(), ok && !v.IsNull() && !v.IsUnknown()
	}
	if name, ok := known("name"); ok {
		if len(name) > 253 || name != strings.ToLower(name) {
			return fmt.Errorf("DNS name requires a lowercase ASCII hostname without a trailing dot")
		}
		for _, label := range strings.Split(name, ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return fmt.Errorf("invalid DNS label")
			}
			for _, ch := range label {
				if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
					return fmt.Errorf("DNS name requires ASCII hostname labels")
				}
			}
		}
	}
	kind, hasKind := known("type")
	if hasKind && !oneOf(kind, "A", "AAAA") {
		return fmt.Errorf("reviewed DNS record types are A and AAAA")
	}
	if text, ok := known("address"); ok {
		ip, err := netip.ParseAddr(text)
		if err != nil || ip.Zone() != "" || ip.String() != text || ip.Is4In6() {
			return fmt.Errorf("DNS address requires a canonical IPv4 or IPv6 address")
		}
		if hasKind && (kind == "A") != ip.Is4() {
			return fmt.Errorf("DNS record type and address family differ")
		}
	}
	return nil
}
