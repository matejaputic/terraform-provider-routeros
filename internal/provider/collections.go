// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/generated"
)

func collectionConstructors() []func() resource.Resource {
	schemas := map[string]func(context.Context) schema.Schema{
		"interface_bridge":       generated.InterfaceBridgeResourceSchema,
		"interface_bridge_port":  generated.InterfaceBridgePortResourceSchema,
		"interface_vlan":         generated.InterfaceVlanResourceSchema,
		"interface_list":         generated.InterfaceListResourceSchema,
		"interface_list_member":  generated.InterfaceListMemberResourceSchema,
		"ip_pool":                generated.IpPoolResourceSchema,
		"ip_dhcp_server_network": generated.IpDhcpServerNetworkResourceSchema,
		"ip_dhcp_server":         generated.IpDhcpServerResourceSchema,
		"ip_dhcp_server_lease":   generated.IpDhcpServerLeaseResourceSchema,
		"ip_route":               generated.IpRouteResourceSchema,
		"ip_firewall_addr_list":  generated.IpFirewallAddrListResourceSchema,
		"ip_firewall_filter":     generated.IpFirewallFilterResourceSchema,
		"ip_firewall_nat":        generated.IpFirewallNatResourceSchema,
		"ip_firewall_mangle":     generated.IpFirewallMangleResourceSchema,
		"ip_firewall_raw":        generated.IpFirewallRawResourceSchema,
		"ip_dhcp_client_option":  generated.IpDhcpClientOptionResourceSchema,
		"ip_dhcp_server_option":  generated.IpDhcpServerOptionResourceSchema,
	}
	result := []func() resource.Resource{}
	for _, policy := range catalog.Collections() {
		result = append(result, func() resource.Resource { return newCollection(policy, schemas[policy.Name](context.Background())) })
	}
	return result
}
