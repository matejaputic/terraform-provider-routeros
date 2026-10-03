// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/generated"
)

func singletonConstructors() []func() resource.Resource {
	bindings := map[string]func(context.Context) schema.Schema{
		"ip_dhcp_server_config":           generated.IpDhcpServerConfigResourceSchema,
		"ip_firewall_connection_tracking": generated.IpFirewallConnectionTrackingResourceSchema,
		"interface_bridge_settings":       generated.InterfaceBridgeSettingsResourceSchema,
		"ip_ipsec_settings":               generated.IpIpsecSettingsResourceSchema,
		"ip_neighbor_discovery_settings":  generated.IpNeighborDiscoverySettingsResourceSchema,
		"ip_settings":                     generated.IpSettingsResourceSchema,
		"ip_tftp_settings":                generated.IpTftpSettingsResourceSchema,
		"ip_traffic_flow":                 generated.IpTrafficFlowResourceSchema,
		"ip_traffic_flow_ipfix":           generated.IpTrafficFlowIpfixResourceSchema,
		"ipv6_settings":                   generated.Ipv6SettingsResourceSchema,
		"ip_nat_pmp":                      generated.IpNatPmpResourceSchema,
		"radius_incoming":                 generated.RadiusIncomingResourceSchema,
		"system_identity":                 generated.SystemIdentityResourceSchema,
		"system_note":                     generated.SystemNoteResourceSchema,
		"system_ntp_client":               generated.SystemNtpClientResourceSchema,
		"system_ntp_server":               generated.SystemNtpServerResourceSchema,
		"tool_mac_server":                 generated.ToolMacServerResourceSchema,
		"tool_mac_server_ping":            generated.ToolMacServerPingResourceSchema,
		"tool_mac_server_winbox":          generated.ToolMacServerWinboxResourceSchema,
	}
	result := []func() resource.Resource{}
	for _, policy := range catalog.Singletons() {
		result = append(result, func() resource.Resource { return newSingleton(policy, bindings[policy.Name](context.Background())) })
	}
	return result
}
