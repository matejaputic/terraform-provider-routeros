# SPDX-License-Identifier: MPL-2.0
"""Explicit reviewed singleton settings subsets, not schema/inventory discovery.

Durations, hardware-specific settings, credentials, dynamic counters and nested
fields not listed here remain unexposed. Runtime uses GET object and fixed POST
/set, despite the pinned publication's synthetic item GET/PATCH observations.
"""
import json
from pathlib import Path
from constructor_identity import constructor

ROOT = Path(__file__).resolve().parents[3]
# field/type, with required suffix !. All others adopt device defaults.
DEFINITIONS = {
 'ip_dhcp_server_config': ('/ip/dhcp-server/config', 'accounting:b radius_password:s'),
 'ip_firewall_connection_tracking': ('/ip/firewall/connection/tracking', 'enabled:s loose_tcp_tracking:b liberal_tcp_tracking:b active_ipv4:bc active_ipv6:bc max_entries:sc'),
 'interface_bridge_settings': ('/interface/bridge/settings', 'allow_fast_path:b use_ip_firewall:b use_ip_firewall_for_pppoe:b use_ip_firewall_for_vlan:b'),
 'ip_ipsec_settings': ('/ip/ipsec/settings', 'accounting:b xauth_use_radius:b'),
 'ip_neighbor_discovery_settings': ('/ip/neighbor/discovery-settings', 'discover_interface_list:s mode:s lldp_mac_phy_config:b lldp_max_frame_size:b lldp_vlan_info:b'),
 'ip_settings': ('/ip/settings', 'accept_redirects:b accept_source_route:b allow_fast_path:b icmp_rate_limit:i rp_filter:s secure_redirects:b send_redirects:b tcp_syncookies:b tcp_timestamps:s'),
 'ip_tftp_settings': ('/ip/tftp/settings', 'max_block_size:i'),
 'ip_traffic_flow': ('/ip/traffic-flow', 'enabled:b interfaces:s packet_sampling:b sampling_interval:i sampling_space:i'),
 'ip_traffic_flow_ipfix': ('/ip/traffic-flow/ipfix', 'bytes:b packets:b src_address:b dst_address:b src_port:b dst_port:b protocol:b tcp_flags:b'),
 'ipv6_settings': ('/ipv6/settings', 'accept_redirects:s accept_router_advertisements:s allow_fast_path:b forward:b multipath_hash_policy:s'),
 'ip_nat_pmp': ('/ip/nat-pmp', 'enabled:b'),
 'radius_incoming': ('/radius/incoming', 'accept:b port:i vrf:s'),
 'system_identity': ('/system/identity', 'name:s!'),
 'system_note': ('/system/note', 'note:s! show_at_login:b show_at_cli_login:b'),
 'system_ntp_client': ('/system/ntp/client', 'enabled:b mode:s vrf:s servers:s status:sc'),
 'system_ntp_server': ('/system/ntp/server', 'enabled:b broadcast:b multicast:b manycast:b use_local_clock:b local_clock_stratum:i vrf:s broadcast_addresses:s'),
 'tool_mac_server': ('/tool/mac-server', 'allowed_interface_list:s!'),
 'tool_mac_server_ping': ('/tool/mac-server/ping', 'enabled:b'),
 'tool_mac_server_winbox': ('/tool/mac-server/mac-winbox', 'allowed_interface_list:s!'),
}
PATHS = {name: definition[0] for name, definition in DEFINITIONS.items()}
CHOICES = {
 ('ip_dhcp_server_config','radius_password'): ['empty','same-as-user'],
 ('ip_firewall_connection_tracking','enabled'): ['auto','yes','no'],
 ('ip_neighbor_discovery_settings','mode'): ['rx-only','tx-only','tx-and-rx'],
 ('ip_settings','rp_filter'): ['no','loose','strict'],
 ('ip_settings','tcp_timestamps'): ['disabled','enabled','random-offset'],
 ('ipv6_settings','accept_redirects'): ['no','yes-if-forwarding-disabled'],
 ('ipv6_settings','accept_router_advertisements'): ['no','yes','yes-if-forwarding-disabled'],
 ('ipv6_settings','multipath_hash_policy'): ['l3','l4','l3-inner'],
 ('system_ntp_client','mode'): ['unicast','broadcast','multicast','manycast'],
}
BOUNDS = {
 ('ip_settings','icmp_rate_limit'): (0,2147483647),
 ('ip_tftp_settings','max_block_size'): (512,65535),
 ('ip_traffic_flow','sampling_interval'): (0,2147483647),
 ('ip_traffic_flow','sampling_space'): (0,2147483647),
 ('radius_incoming','port'): (1,65535),
 ('system_ntp_server','local_clock_stratum'): (1,16),
}

def policies():
 result=[]
 for name,(path,fields) in DEFINITIONS.items():
  attributes=[]
  for token in ['id:sc']+fields.split():
   field,kind=token.split(':');mode='required' if kind.endswith('!') else 'computed' if kind.endswith('c') else 'computed_optional'
   typ={'s':'string','b':'boolean','i':'integer'}[kind[0]]
   attr={'name':field,'wire':'.id' if field=='id' else field.replace('_','-'),'type':typ,'mode':mode,'force_new':False,'sensitive':False,'codec':{'string':'wire-string','boolean':'yes/no','integer':'decimal'}[typ], 'enum_kind':'none','validators':'maintained singleton validation' if (name,field) in CHOICES or (name,field) in BOUNDS else None,'default':'server-owned' if mode=='computed_optional' else None,'provenance':'explicit singleton subset reviewed against pinned SDK declarations and published item PATCH structure; object/set lifecycle maintained separately'}
   if (name,field) in CHOICES: attr['choices']=CHOICES[name,field]
   if (name,field) in BOUNDS: attr['minimum'],attr['maximum']=BOUNDS[name,field]
   attributes.append(attr)
  result.append({'revision':'singletons-curated-v1','reference_constructor':constructor(name),'resource_name':name,'wire_path':path,'lifecycle':'singleton','schema_version':0,'migration_compatibility':'not claimed','reference_sha':'0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb','attributes':attributes,'required':[f['name'] for f in attributes if f['mode']=='required'],'optional_computed':[f['name'] for f in attributes if f['mode']=='computed_optional'],'computed':[f['name'] for f in attributes if f['mode']=='computed'],'booleans':[f['name'] for f in attributes if f['type']=='boolean']})
 return result

if __name__=='__main__':
 (ROOT/'internal/catalog/singletons.json').write_text(json.dumps(policies(),indent=2)+'\n')
