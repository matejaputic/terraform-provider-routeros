#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Reproduce the reviewed initial networking policy, not discover authorization."""
import json
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
# (wire, type, mode). This is intentionally a reviewed field subset.
resources={
 'interface_bridge':('/interface/bridge',[('name','string','required'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('protocol-mode','string','computed_optional'),('vlan-filtering','boolean','computed_optional'),('pvid','integer','computed_optional'),('running','boolean','computed')]),
 'interface_bridge_port':('/interface/bridge/port',[('bridge','string','required'),('interface','string','required'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('pvid','integer','computed_optional'),('edge','string','computed_optional'),('hw','boolean','computed_optional'),('dynamic','boolean','computed'),('inactive','boolean','computed')]),
 'interface_vlan':('/interface/vlan',[('name','string','required'),('interface','string','required'),('vlan-id','integer','required'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('use-service-tag','boolean','computed_optional'),('mtu','integer','computed_optional'),('running','boolean','computed')]),
 'interface_list':('/interface/list',[('name','string','required'),('comment','string','computed_optional'),('include','string','computed_optional'),('exclude','string','computed_optional'),('builtin','boolean','computed'),('dynamic','boolean','computed')]),
 'interface_list_member':('/interface/list/member',[('interface','string','required'),('list','string','required'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('dynamic','boolean','computed')]),
 'ip_pool':('/ip/pool',[('name','string','required'),('ranges','array','required'),('comment','string','computed_optional'),('next-pool','string','computed_optional')]),
 'ip_dhcp_server_network':('/ip/dhcp-server/network',[('address','string','required'),('gateway','string','computed_optional'),('dns-server','array','computed_optional'),('dns-none','boolean','computed_optional'),('netmask','integer','computed_optional'),('domain','string','computed_optional'),('comment','string','computed_optional'),('dynamic','boolean','computed')]),
 'ip_dhcp_server':('/ip/dhcp-server',[('name','string','required'),('interface','string','required'),('address-pool','string','required'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('add-arp','boolean','computed_optional'),('conflict-detection','boolean','computed_optional'),('invalid','boolean','computed'),('dynamic','boolean','computed')]),
 'ip_dhcp_server_lease':('/ip/dhcp-server/lease',[('address','string','required'),('mac-address','string','required'),('server','string','computed_optional'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('block-access','boolean','computed_optional'),('dynamic','boolean','computed'),('status','string','computed')]),
 'ip_route':('/ip/route',[('dst-address','string','required'),('gateway','string','required'),('distance','integer','computed_optional'),('routing-table','string','computed_optional'),('scope','integer','computed_optional'),('target-scope','integer','computed_optional'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('dynamic','boolean','computed'),('active','boolean','computed'),('static','boolean','computed')]),
 'ip_firewall_addr_list':('/ip/firewall/address-list',[('list','string','required'),('address','string','required'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('dynamic','boolean','computed'),('creation-time','string','computed')]),
 'ip_firewall_filter':('/ip/firewall/filter',[('chain','string','required'),('action','string','required'),('src-address-list','string','computed_optional'),('dst-address-list','string','computed_optional'),('comment','string','computed_optional'),('disabled','boolean','computed_optional'),('log','boolean','computed_optional'),('dynamic','boolean','computed'),('invalid','boolean','computed'),('place-before','string','computed_optional')]),
}
# Each rule family is separately reviewed; do not discover registration.
base=resources['ip_firewall_filter'][1]
resources['ip_firewall_nat']=('/ip/firewall/nat',base+[('to-addresses','string','computed_optional')])
resources['ip_firewall_mangle']=('/ip/firewall/mangle',base+[('new-connection-mark','string','computed_optional'),('new-packet-mark','string','computed_optional'),('passthrough','boolean','computed_optional')])
resources['ip_firewall_raw']=('/ip/firewall/raw',list(base))
# Reviewed DHCP option wave; no aliases or automatically discovered fields.
option_fields=[('name','string','required'),('code','integer','required'),('value','string','required'),('raw-value','string','computed')]
resources['ip_dhcp_client_option']=('/ip/dhcp-client/option',list(option_fields))
resources['ip_dhcp_server_option']=('/ip/dhcp-server/option',option_fields+[('comment','string','computed_optional'),('force','boolean','computed_optional')])
# Reviewed DHCP relay: deliberately omit duration and option-82 payload fields
# until their canonicalization and clearing semantics have dedicated review.
resources['ip_dhcp_relay']=('/ip/dhcp-relay',[
 ('name','string','required'),('interface','string','required'),
 ('dhcp-server','string','required'),('disabled','boolean','computed_optional'),
 ('local-address','string','computed_optional'),('add-relay-info','boolean','computed_optional'),
 ('invalid','boolean','computed')])
# Reviewed named address DNS records: A/AAAA only; no regex, forwarding,
# dynamic firewall address-list actions, or duration canonicalization.
resources['ip_dns_record']=('/ip/dns/static',[
 ('name','string','required'),('type','string','required'),('address','string','required'),
 ('comment','string','computed_optional'),('disabled','boolean','computed_optional'),
 ('match-subdomain','boolean','computed_optional'),('dynamic','boolean','computed')])
policies=[]
for name,(path,items) in resources.items():
 attrs=[]
 for wire,kind,mode in [('.id','string','computed')]+items:
  tf='id' if wire=='.id' else wire.replace('-','_')
  attrs.append({'name':tf,'wire':wire,'type':kind,'mode':mode,'sensitive':False,'force_new':wire=='name','enum_kind':'device-instance-removed' if wire in ('interface','bridge','list','include','exclude','next-pool') else 'none','validators':None,'default':'server-owned' if mode=='computed_optional' else None,'codec':'csv' if kind=='array' else 'yes/no' if kind=='boolean' else 'decimal' if kind=='integer' else 'wire-string','provenance':'reviewed pinned SDK schema and immutable upstream; live evidence tracked separately'})
  if name=='ip_dhcp_relay' and tf in ('dhcp_server','local_address'): attrs[-1]['validators']='maintained ValidateConfig: validateDHCPRelayString'
  if name=='ip_dns_record' and tf in ('name','type','address'): attrs[-1]['validators']='maintained ValidateConfig: validateDNSAddressRecord'
  if name=='ip_dns_record' and tf=='type': attrs[-1]['force_new']=True
  if name=='ip_dns_record' and tf=='match_subdomain': attrs[-1]['read_default']='false'
  if name in ('ip_firewall_filter','ip_firewall_nat','ip_firewall_mangle','ip_firewall_raw') and tf=='place_before': attrs[-1]['codec']='placement'
  if name in ('ip_firewall_filter','ip_firewall_nat','ip_firewall_mangle','ip_firewall_raw') and tf in ('src_address_list','dst_address_list','to_addresses','new_connection_mark','new_packet_mark'): attrs[-1]['read_default']=''
  if tf in ('comment','include','exclude'): attrs[-1]['read_default']=''
  if tf=='next_pool': attrs[-1]['read_default']='none'
  if name=='ip_dhcp_server_network' and tf in ('dns_server','domain'): attrs[-1]['read_default']=''
  if name=='ip_dhcp_server_network' and tf=='dns_none': attrs[-1]['read_default']='false'
  if name=='ip_dhcp_server' and tf=='add_arp': attrs[-1]['read_default']='false'
  if name=='ip_dhcp_server' and tf=='conflict_detection': attrs[-1]['read_default']='true'
  if name=='ip_dhcp_server_lease' and tf=='block_access': attrs[-1]['read_default']='false'
  if name=='ip_route' and tf in ('dynamic','active'): attrs[-1]['read_default']='false'
  if name=='ip_dhcp_server_option' and tf=='force': attrs[-1]['read_default']='false'
  if name=='interface_bridge' and tf=='pvid': attrs[-1]['conditional_read']='vlan_filtering'
 policy={'revision':'collections-curated-v5','resource_name':name,'wire_path':path,'schema_version':0,'migration_compatibility':'not claimed','attributes':attrs,'reference_sha':'0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb','required':[f['name'] for f in attrs if f['mode']=='required'],'optional_computed':[f['name'] for f in attrs if f['mode']=='computed_optional'],'computed':[f['name'] for f in attrs if f['mode']=='computed'],'booleans':[f['name'] for f in attrs if f['type']=='boolean']}
 policies.append(policy)
(ROOT/'internal/catalog/collections.json').write_text(json.dumps(policies,indent=2)+'\n')
