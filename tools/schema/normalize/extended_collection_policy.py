# SPDX-License-Identifier: MPL-2.0
"""Reviewed scalar/CSV collection subsets. Not extracted exposure authority.

Token flags: ! required, ^ replacement, ~ sensitive, * computed, + CSV string set.
Constraints after / are implemented in maintained lifecycle code, not an emitter.
Unlisted SDK fields/actions/upgraders remain unsupported. Aliases are not exposed.
"""
import json
from pathlib import Path
from constructor_identity import constructor
ROOT=Path(__file__).resolve().parents[3]
DEFINITIONS={
 'system_certificate_scep_server':('/certificate/scep-server','ca-cert:s! path:s! days-valid:i disabled:b'),
 'ip_dhcp_client':('/ip/dhcp-client','interface:s! disabled:b comment:s add-default-route:s use-peer-dns:b use-peer-ntp:b status:s* invalid:b*'),
 'ip_dhcp_server_option_matcher':('/ip/dhcp-server/matcher','name:s!^ server:s! address-pool:s! code:i! value:s! matching-type:s! disabled:b comment:s'),
 'ip_dhcp_server_option_set':('/ip/dhcp-server/option/sets','name:s!^ options:s!+ comment:s'),
 'ip_dns_adlist':('/ip/dns/adlist','url:s!~/url disabled:b! ssl-verify:b'),
 'ip_vrf':('/ip/vrf','name:s!^ interfaces:s!+ comment:s'),
 'ipv6_address':('/ipv6/address','address:s!/address6 interface:s! advertise:b disabled:b comment:s dynamic:b* invalid:b*'),
 'ipv6_dhcp_client':('/ipv6/dhcp-client','interface:s! request:s! pool-name:s! pool-prefix-length:i disabled:b comment:s add-default-route:b use-peer-dns:b status:s* invalid:b*'),
 'ipv6_dhcp_client_option':('/ipv6/dhcp-client/option','name:s!^ code:i! value:s! raw-value:s*'),
 'ipv6_firewall_addr_list':('/ipv6/firewall/address-list','list:s! address:s!/ip6_or_prefix disabled:b comment:s dynamic:b*'),
 'ipv6_firewall_filter':('/ipv6/firewall/filter','chain:s! action:s! src-address-list:s dst-address-list:s disabled:b comment:s log:b dynamic:b* invalid:b*'),
 'ipv6_neighbor_discovery':('/ipv6/nd','interface:s! disabled:b advertise-dns:s advertise-mac-address:b managed-address-configuration:b other-configuration:b hop-limit:i mtu:i invalid:b*'),
 'ipv6_route':('/ipv6/route','dst-address:s!/prefix6 gateway:s! distance:i routing-table:s disabled:b comment:s dynamic:b* static:b* active:b*'),
 'interface_6to4':('/interface/6to4','name:s!^ local-address:s!/ip4 remote-address:s!/ip4 mtu:i clamp-tcp-mss:b disabled:b comment:s running:b*'),
 'interface_bonding':('/interface/bonding','name:s!^ slaves:s!+ mode:s primary:s mtu:i disabled:b comment:s running:b*'),
 'interface_bridge_vlan':('/interface/bridge/vlan','bridge:s! vlan-ids:s!/vlan_ids tagged:s+ untagged:s+ disabled:b comment:s dynamic:b*'),
 'interface_dot1x_client':('/interface/dot1x/client','interface:s! identity:s! password:s~ eap-methods:s+ disabled:b comment:s status:s*'),
 'interface_dot1x_server':('/interface/dot1x/server','interface:s! auth-types:s+ accounting:b disabled:b comment:s'),
 'interface_gre6':('/interface/gre6','name:s!^ local-address:s!/ip6 remote-address:s!/ip6 mtu:i clamp-tcp-mss:b disabled:b comment:s running:b*'),
 'interface_l2tp_client':('/interface/l2tp-client','name:s!^ connect-to:s! user:s! password:s~ profile:s use-peer-dns:s add-default-route:b disabled:b comment:s running:b*'),
 'interface_macvlan':('/interface/macvlan','name:s!^ interface:s! mode:s mtu:i disabled:b comment:s running:b*'),
 'interface_ovpn_server':('/interface/ovpn-server','name:s!^ user:s! disabled:b comment:s running:b*'),
 'interface_pppoe_client':('/interface/pppoe-client','name:s!^ interface:s! user:s! password:s~ profile:s use-peer-dns:b add-default-route:b disabled:b comment:s running:b*'),
 'interface_pppoe_server':('/interface/pppoe-server','name:s!^ user:s! service:s! disabled:b comment:s running:b*'),
 'interface_sstp_client':('/interface/sstp-client','name:s!^ connect-to:s! user:s! password:s~ port:i profile:s verify-server-certificate:b add-default-route:b disabled:b comment:s running:b*'),
 'interface_veth':('/interface/veth','name:s!^ address:s!+ gateway:s/ip4 dhcp:b disabled:b comment:s running:b*'),
 'interface_vxlan':('/interface/vxlan','name:s!^ vni:i! local-address:s!/ip4 port:i mtu:i disabled:b comment:s running:b*'),
 'interface_vxlan_vteps':('/interface/vxlan/vteps','interface:s! remote-ip:s!/ip4 disabled:b comment:s'),
 'interface_wireguard':('/interface/wireguard','name:s!^ listen-port:i mtu:i disabled:b comment:s public-key:s*'),
 'interface_wireguard_peer':('/interface/wireguard/peers','interface:s! public-key:s! allowed-address:s!+ endpoint-address:s endpoint-port:i persistent-keepalive:s/duration disabled:b comment:s'),
 'ip_dns_forwarders':('/ip/dns/forwarders','name:s!^ dns-servers:s!+ disabled:b comment:s'),
 'ip_firewall_layer7_protocol':('/ip/firewall/layer7-protocol','name:s!^ regexp:s!'),
 'ip_hotspot':('/ip/hotspot','name:s!^ interface:s! address-pool:s! profile:s! addresses-per-mac:i disabled:b invalid:b*'),
 'ip_hotspot_ip_binding':('/ip/hotspot/ip-binding','server:s! address:s!/ip4 to-address:s/ip4 type:s! mac-address:s/mac disabled:b comment:s'),
 'ip_hotspot_profile':('/ip/hotspot/profile','name:s!^ hotspot-address:s/ip4 dns-name:s login-by:s+ use-radius:b'),
 'ip_hotspot_user':('/ip/hotspot/user','name:s!^ server:s! profile:s! password:s~ disabled:b comment:s'),
 'ip_hotspot_user_profile':('/ip/hotspot/user/profile','name:s!^ shared-users:i add-mac-cookie:b transparent-proxy:b'),
 'ip_hotspot_walled_garden':('/ip/hotspot/walled-garden','server:s! dst-host:s! action:s method:s path:s disabled:b comment:s'),
 'ip_hotspot_walled_garden_ip':('/ip/hotspot/walled-garden/ip','server:s! dst-address:s!/prefix4 action:s! protocol:s dst-port:s disabled:b comment:s'),
 'ip_ipsec_identity':('/ip/ipsec/identity','peer:s! auth-method:s secret:s~ disabled:b comment:s dynamic:b*'),
 'ip_ipsec_mode_config':('/ip/ipsec/mode-config','name:s!^ address-pool:s! responder:b split-include:s+'),
 'ip_ipsec_peer':('/ip/ipsec/peer','name:s!^ address:s!/prefix4 profile:s! exchange-mode:s passive:b disabled:b comment:s dynamic:b*'),
 'ip_ipsec_policy':('/ip/ipsec/policy','src-address:s!/prefix dst-address:s!/prefix template:b group:s! proposal:s! action:s disabled:b comment:s dynamic:b* invalid:b*'),
 'ip_ipsec_policy_group':('/ip/ipsec/policy/group','name:s!^'),
 'ip_ipsec_profile':('/ip/ipsec/profile','name:s!^ dh-group:s!+ enc-algorithm:s!+ hash-algorithm:s! nat-traversal:b'),
 'ip_ipsec_proposal':('/ip/ipsec/proposal','name:s!^ auth-algorithms:s!+ enc-algorithms:s!+ pfs-group:s disabled:b comment:s'),
 'ip_tftp':('/ip/tftp','req-filename:s! real-filename:s! read-only:b allow:b disabled:b'),
 'ip_traffic_flow_target':('/ip/traffic-flow/target','dst-address:s!/ip4 port:i! version:s! src-address:s/ip4 disabled:b'),
 'ipv6_dhcp_server':('/ipv6/dhcp-server','name:s!^ interface:s! prefix-pool:s! address-pool:s preference:i disabled:b comment:s dynamic:b* invalid:b*'),
 'ipv6_dhcp_server_option':('/ipv6/dhcp-server/option','name:s!^ code:i! value:s! comment:s raw-value:s*'),
 'ipv6_dhcp_server_option_sets':('/ipv6/dhcp-server/option/sets','name:s!^ options:s!+ comment:s'),
 'ipv6_nd_prefix':('/ipv6/nd/prefix','interface:s! prefix:s!/prefix6 autonomous:b on-link:b disabled:b comment:s invalid:b*'),
 'ipv6_pool':('/ipv6/pool','name:s!^ prefix:s!/prefix6 prefix-length:i! dynamic:b*'),
 'ip_nat_pmp_interfaces':('/ip/nat-pmp/interfaces','interface:s! type:s! forced-ip:s/ip4 disabled:b dynamic:b*'),
 'interface_ovpn_client':('/interface/ovpn-client','name:s!^ connect-to:s! user:s! password:s~ port:i profile:s protocol:s verify-server-certificate:b use-peer-dns:s add-default-route:b disabled:b comment:s running:b*'),
 'ppp_profile':('/ppp/profile','name:s!^ local-address:s! remote-address:s! change-tcp-mss:s use-encryption:s only-one:s comment:s'),
 'ppp_secret':('/ppp/secret','name:s!^ password:s~ profile:s! service:s local-address:s/ip4 remote-address:s/ip4 disabled:b comment:s'),
 'queue_simple':('/queue/simple','name:s!^ target:s!+ max-limit:s/rate_pair limit-at:s/rate_pair priority:s/priority_pair parent:s disabled:b comment:s dynamic:b* invalid:b*'),
 'queue_tree':('/queue/tree','name:s!^ parent:s! packet-mark:s! max-limit:i limit-at:i priority:i disabled:b comment:s dynamic:b* invalid:b*'),
 'queue_type':('/queue/type','name:s!^ kind:s!^ pfifo-limit:i'),
 'radius':('/radius','address:s!/ip service:s!+ protocol:s secret:s~ authentication-port:i accounting-port:i disabled:b comment:s'),
 'routing_bfd_configuration':('/routing/bfd/configuration','interfaces:s!+ addresses:s+ forbid-bfd:b multiplier:i disabled:b'),
 'routing_bgp_connection':('/routing/bgp/connection','name:s!^ instance:s! remote.address:s!/prefix local.role:s! remote.as:i connect:b listen:b disabled:b comment:s'),
 'routing_bgp_evpn':('/routing/bgp/evpn','name:s!^ instance:s! rd:s! vni:i! vrf:s! disabled:b comment:s'),
 'routing_bgp_instance':('/routing/bgp/instance','name:s!^ as:i! router-id:s!/ip4 routing-table:s vrf:s disabled:b comment:s'),
 'routing_bgp_template':('/routing/bgp/template','name:s!^ as:i! afi:s!+ disabled:b comment:s'),
 'routing_bgp_vpn':('/routing/bgp/vpn','name:s!^ instance:s! route-distinguisher:s! vrf:s! label-allocation-policy:s! disabled:b comment:s'),
 'routing_filter_rule':('/routing/filter/rule','chain:s! rule:s! disabled:b comment:s inactive:b*'),
 'routing_id':('/routing/id','name:s!^ id:s!/ip4 disabled:b comment:s dynamic:b*'),
 'routing_igmp_proxy_interface':('/routing/igmp-proxy/interface','interface:s! upstream:b alternative-subnets:s+ threshold:i disabled:b comment:s'),
 'routing_ospf_area':('/routing/ospf/area','name:s!^ instance:s! area-id:s!/ip4 type:s disabled:b comment:s inactive:b*'),
 'routing_ospf_area_range':('/routing/ospf/area/range','area:s! prefix:s!/prefix cost:i advertise:b disabled:b comment:s'),
 'routing_ospf_instance':('/routing/ospf/instance','name:s!^ version:i router-id:s!/ip4 routing-table:s vrf:s disabled:b comment:s inactive:b*'),
 'routing_ospf_interface_template':('/routing/ospf/interface-template','area:s! interfaces:s!+ networks:s+ cost:i priority:i disabled:b comment:s inactive:b*'),
 'routing_rule':('/routing/rule','dst-address:s!/prefix4 table:s! action:s! disabled:b comment:s inactive:b*'),
 'snmp_community':('/snmp/community','name:s!^~ addresses:s!+ security:s read-access:b write-access:b disabled:b comment:s'),
 'system_logging':('/system/logging','action:s! topics:s!+ prefix:s disabled:b'),
 'system_scheduler':('/system/scheduler','name:s!^ on-event:s!~ interval:s/duration start-time:s disabled:b! comment:s run-count:i*'),
 'tool_graphing_interface':('/tool/graphing/interface','interface:s! allow-address:s/prefix4 store-on-disk:b disabled:b'),
 'tool_graphing_queue':('/tool/graphing/queue','simple-queue:s! allow-address:s/prefix4 allow-target:b store-on-disk:b disabled:b'),
 'tool_graphing_resource':('/tool/graphing/resource','allow-address:s!/prefix4 store-on-disk:b disabled:b'),
 'tool_netwatch':('/tool/netwatch','name:s!^ host:s!/ip type:s interval:s/duration disabled:b! comment:s'),
 'ip_upnp_interfaces':('/ip/upnp/interfaces','interface:s! type:s! forced-ip:s/ip4 disabled:b dynamic:b*'),
 'system_user':('/user','name:s!^ group:s! password:s!~ disabled:b! comment:s'),
 'system_user_group':('/user/group','name:s!^ policy:s!+/user_policy comment:s'),
 'system_user_sshkeys':('/user/ssh-keys','user:s!^ key:s!^~ fingerprint:s* key-owner:s*~ key-type:s* bits:i*'),
}
NAME_ALIASES = {('routing_id','id'):'router_id', ('routing_bgp_template','afi'):'address_families'}
CHOICES={
 ('ipv6_neighbor_discovery','advertise_dns'):['yes','no','yes-pref64'],
 ('interface_l2tp_client','use_peer_dns'):['yes','no'],
 ('interface_ovpn_client','use_peer_dns'):['yes','no'],
 ('ip_dhcp_client','add_default_route'):['yes','no','special-classless'],
 ('ip_dhcp_server_option_matcher','matching_type'):['exact','substring'],
 ('ipv6_dhcp_client','request'):['address','prefix','address,prefix'],
 ('ipv6_firewall_filter','action'):['accept','drop','log','passthrough','return'],
 ('interface_bonding','mode'):['802.3ad','active-backup','balance-alb','balance-rr','balance-tlb','balance-xor','broadcast'],
 ('interface_macvlan','mode'):['bridge','private'],
 ('ip_hotspot_ip_binding','type'):['regular','bypassed','blocked'],
 ('ip_hotspot_walled_garden','action'):['allow','deny'],
 ('ip_hotspot_walled_garden_ip','action'):['accept','drop','reject'],
 ('ip_ipsec_identity','auth_method'):['pre-shared-key'],
 ('ip_ipsec_peer','exchange_mode'):['ike2','main','aggressive'],
 ('ip_ipsec_policy','action'):['encrypt'],
 ('ip_ipsec_proposal','pfs_group'):['none','modp2048','modp3072','modp4096','ecp256','ecp384'],
 ('ip_traffic_flow_target','version'):['5','9','IPFIX'],
 ('ip_nat_pmp_interfaces','type'):['internal','external'],
 ('interface_ovpn_client','protocol'):['tcp','udp'],
 ('ppp_secret','service'):['any','async','isdn','l2tp','ovpn','pppoe','pptp','sstp'],
 ('ppp_profile','change_tcp_mss'):['yes','no','default'],
 ('ppp_profile','use_encryption'):['yes','no','required','default'],
 ('ppp_profile','only_one'):['yes','no','default'],
 ('queue_type','kind'):['pfifo'],
 ('radius','protocol'):['udp'],
 ('routing_bgp_vpn','label_allocation_policy'):['per-vrf','per-prefix'],
 ('routing_bgp_connection','local_role'):['ibgp','ebgp','ibgp-rr','ebgp-customer','ebgp-provider','ebgp-peer','ebgp-rs','ebgp-rs-client'],
 ('routing_ospf_area','type'):['default','stub','nssa'],
 ('routing_rule','action'):['lookup','lookup-only-in-table'],
 ('snmp_community','security'):['none'],
 ('tool_netwatch','type'):['simple'],
 ('ip_upnp_interfaces','type'):['internal','external'],
}
BOUNDS={
 'code':(1,65535),'port':(1,65535),'listen_port':(1,65535),'endpoint_port':(1,65535),'authentication_port':(1,65535),'accounting_port':(1,65535),'mtu':(68,65535),'vni':(1,16777215),'prefix_length':(1,128),'pool_prefix_length':(1,128),'as':(1,4294967295),'remote_as':(1,4294967295),'cost':(0,65535),'pfifo_limit':(1,4294967295),'multiplier':(1,255),'threshold':(1,255),'hop_limit':(0,255),'preference':(0,255),'days_valid':(1,3650),'shared_users':(1,2147483647),'addresses_per_mac':(1,2147483647),'max_limit':(0,2147483647),'limit_at':(0,2147483647),'priority':(1,8),'distance':(1,255),'version':(2,3),
}

def policies():
 result=[]
 for name,(path,definition) in DEFINITIONS.items():
  fields=[]
  for token in ['.id:s*']+definition.split():
   wire,flags=token.split(':');kind=flags[0];flags,_,constraint=flags.partition('/')
   tf='id' if wire=='.id' else NAME_ALIASES.get((name,wire),wire.replace('-','_').replace('.','_'))
   typ={'s':'string','i':'integer','b':'boolean'}[kind];mode='required' if '!' in flags else 'computed' if '*' in flags else 'computed_optional'
   field={'name':tf,'wire':wire,'type':typ,'mode':mode,'sensitive':'~' in flags,'force_new':'^' in flags,'codec':'csv-string-set' if '+' in flags else 'routeros-duration' if constraint=='duration' else {'string':'wire-string','boolean':'yes/no','integer':'decimal'}[typ],'enum_kind':'none','validators':'maintained extended collection validation','default':'server-owned' if mode=='computed_optional' else None,'provenance':'explicit reviewed extended collection scalar/CSV subset; published structures are separately verified, not lifecycle authority'}
   if constraint:field['constraint']=constraint
   if (name,tf) in CHOICES:field['choices']=CHOICES[name,tf]
   if typ=='integer' and tf in BOUNDS:field['minimum'],field['maximum']=BOUNDS[tf]
   if name in ('interface_6to4','interface_gre6','ipv6_neighbor_discovery') and tf in ('mtu','hop_limit'):field['codec']='auto-decimal'
   if name=='interface_wireguard_peer' and tf=='endpoint_port':field['minimum']=0
   if name=='routing_ospf_interface_template' and tf=='priority':field['minimum'],field['maximum']=0,255
   if name=='ip_dhcp_server_option_matcher' and tf=='code':field['maximum']=254
   if tf=='comment' or '+' in flags and mode=='computed_optional' or tf=='prefix' and name=='system_logging':field['read_default']=''
   if tf=='disabled':field['read_default']='false'
   if '~' in flags and tf in ('password','secret','key'):field['preserve_secret_on_omission']=True
   fields.append(field)
  policy={'revision':'extended-collections-curated-v1','reference_constructor':constructor(name),'resource_name':name,'wire_path':path,'schema_version':0,'migration_compatibility':'not claimed','attributes':fields,'reference_sha':'0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb','required':[f['name'] for f in fields if f['mode']=='required'],'optional_computed':[f['name'] for f in fields if f['mode']=='computed_optional'],'computed':[f['name'] for f in fields if f['mode']=='computed'],'booleans':[f['name'] for f in fields if f['type']=='boolean']}
  if name in ('system_user_sshkeys','ip_ipsec_policy_group'):policy['replacement_only']=True
  if name=='interface_veth':policy['required_package']='container'
  result.append(policy)
 return result

if __name__=='__main__':
 (ROOT/'internal/catalog/extended-collections.json').write_text(json.dumps(policies(),indent=2)+'\n')
