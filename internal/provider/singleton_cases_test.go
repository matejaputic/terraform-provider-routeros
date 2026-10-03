// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
)

// Explicit safe configuration cases, independent of generator inputs. No scripts,
// credentials, management IP forwarding changes, enabled exporters or NTP fetch.
var singletonCases = map[string][2]map[string]any{
	"ip_dhcp_server_config":           {{"accounting": false, "radius_password": "empty"}, {"accounting": true, "radius_password": "same-as-user"}},
	"ip_firewall_connection_tracking": {{"enabled": "auto", "loose_tcp_tracking": true, "liberal_tcp_tracking": false}, {"enabled": "auto", "loose_tcp_tracking": false, "liberal_tcp_tracking": false}},
	"interface_bridge_settings":       {{"allow_fast_path": true, "use_ip_firewall": false, "use_ip_firewall_for_pppoe": false, "use_ip_firewall_for_vlan": false}, {"allow_fast_path": false, "use_ip_firewall": false, "use_ip_firewall_for_pppoe": false, "use_ip_firewall_for_vlan": false}},
	"ip_ipsec_settings":               {{"accounting": false, "xauth_use_radius": false}, {"accounting": true, "xauth_use_radius": false}},
	"ip_neighbor_discovery_settings":  {{"discover_interface_list": "none", "mode": "rx-only"}, {"discover_interface_list": "none", "mode": "tx-only"}},
	"ip_settings":                     {{"icmp_rate_limit": 10, "accept_source_route": false, "rp_filter": "no"}, {"icmp_rate_limit": 20, "accept_source_route": false, "rp_filter": "no"}},
	"ip_tftp_settings":                {{"max_block_size": 4096}, {"max_block_size": 8192}},
	"ip_traffic_flow":                 {{"enabled": false, "interfaces": "tf-test", "packet_sampling": false, "sampling_interval": 100, "sampling_space": 100}, {"enabled": false, "interfaces": "tf-test", "packet_sampling": true, "sampling_interval": 200, "sampling_space": 200}},
	"ip_traffic_flow_ipfix":           {{"bytes": true, "packets": true, "src_address": true, "dst_address": true, "src_port": true, "dst_port": true, "protocol": true, "tcp_flags": true}, {"bytes": false, "packets": false, "src_address": true, "dst_address": true, "src_port": true, "dst_port": true, "protocol": true, "tcp_flags": false}},
	"ipv6_settings":                   {{"allow_fast_path": true, "accept_redirects": "no", "accept_router_advertisements": "no", "multipath_hash_policy": "l3"}, {"allow_fast_path": false, "accept_redirects": "no", "accept_router_advertisements": "no", "multipath_hash_policy": "l4"}},
	"ip_nat_pmp":                      {{"enabled": false}, {"enabled": true}},
	"radius_incoming":                 {{"accept": false, "port": 3799, "vrf": "main"}, {"accept": false, "port": 3800, "vrf": "main"}},
	"system_identity":                 {{"name": "tf-settings-initial"}, {"name": "tf-settings-updated"}},
	"system_note":                     {{"note": "Owned disposable settings test\nInitial note", "show_at_login": false, "show_at_cli_login": false}, {"note": "", "show_at_login": false, "show_at_cli_login": false}},
	"system_ntp_client":               {{"enabled": false, "mode": "unicast", "vrf": "main", "servers": ""}, {"enabled": false, "mode": "broadcast", "vrf": "main", "servers": ""}},
	"system_ntp_server":               {{"enabled": false, "broadcast": false, "multicast": false, "manycast": false, "use_local_clock": false, "local_clock_stratum": 5, "vrf": "main", "broadcast_addresses": ""}, {"enabled": false, "broadcast": false, "multicast": false, "manycast": false, "use_local_clock": false, "local_clock_stratum": 6, "vrf": "main", "broadcast_addresses": ""}},
	"tool_mac_server":                 {{"allowed_interface_list": "none"}, {"allowed_interface_list": "static"}},
	"tool_mac_server_ping":            {{"enabled": false}, {"enabled": true}},
	"tool_mac_server_winbox":          {{"allowed_interface_list": "none"}, {"allowed_interface_list": "static"}},
}

func singletonConfig(provider string, changed bool) string {
	var b strings.Builder
	b.WriteString(provider)
	index := 0
	if changed {
		index = 1
	}
	for _, p := range catalog.Singletons() {
		fmt.Fprintf(&b, "\nresource %q %q {\n", "routeros_"+p.Name, "test")
		values := singletonCases[p.Name][index]
		keys := []string{}
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			encoded, _ := json.Marshal(values[key])
			fmt.Fprintf(&b, " %s = %s\n", key, encoded)
		}
		b.WriteString("}\n")
	}
	return b.String()
}
