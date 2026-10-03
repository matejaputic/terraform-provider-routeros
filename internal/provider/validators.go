// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"net"
)

type ipValidator struct{ cidr bool }

func (v ipValidator) Description(context.Context) string {
	if v.cidr {
		return "Must be an IP address or CIDR."
	}
	return "Must be an IP address."
}
func (v ipValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }
func (v ipValidator) ValidateString(ctx context.Context, q validator.StringRequest, r *validator.StringResponse) {
	if q.ConfigValue.IsNull() || q.ConfigValue.IsUnknown() {
		return
	}
	s := q.ConfigValue.ValueString()
	if net.ParseIP(s) != nil {
		return
	}
	if v.cidr {
		if _, _, e := net.ParseCIDR(s); e == nil {
			return
		}
	}
	r.Diagnostics.AddAttributeError(q.Path, "Invalid IP address", v.Description(ctx))
}

type nonemptyValidator struct{}

func (nonemptyValidator) Description(context.Context) string               { return "Must be nonempty." }
func (v nonemptyValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }
func (v nonemptyValidator) ValidateString(ctx context.Context, q validator.StringRequest, r *validator.StringResponse) {
	if !q.ConfigValue.IsNull() && !q.ConfigValue.IsUnknown() && q.ConfigValue.ValueString() == "" {
		r.Diagnostics.AddAttributeError(q.Path, "Empty interface", v.Description(ctx))
	}
}
