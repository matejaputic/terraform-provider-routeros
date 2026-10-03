// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/generated"
)

// recoveryModel retains known configured values and the created ID while making
// unresolved computed values null, so an errored post-create read is recoverable.
func recoveryModel(m generated.IpAddressModel) generated.IpAddressModel {
	for _, v := range []*types.String{&m.ActualInterface, &m.Comment, &m.Network, &m.Vrf} {
		if v.IsUnknown() {
			*v = types.StringNull()
		}
	}
	for _, v := range []*types.Bool{&m.Disabled, &m.Dynamic, &m.Invalid, &m.Slave} {
		if v.IsUnknown() {
			*v = types.BoolNull()
		}
	}
	return m
}
