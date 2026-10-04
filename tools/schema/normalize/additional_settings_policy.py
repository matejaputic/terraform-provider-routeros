#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Explicit reviewed Remaining resource coverage contracts; neither the ledger nor inventory exposes fields.

Settings use path identity/GET object/fixed POST set/unmanage destroy. Hardware,
commands, multi-object OpenVPN settings and asynchronous resources require their
own review and are deliberately absent here. No SDK code is executed.
"""
import json
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
REFERENCE='0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb'
# (reference identity, fixed path, explicitly selected scalar/CSV fields, package)
SETTINGS={
 'capsman_aaa':('ResourceCapsManAaa','/caps-man/aaa','called-format:s mac-format:s mac-mode:s','wireless'),
 'capsman_manager':('ResourceCapsManManager','/caps-man/manager','enabled:b require-peer-certificate:b certificate:s ca-certificate:s upgrade-policy:s','wireless'),
 'container_config':('ResourceContainerConfig','/container/config','registry-url:s username:s~ password:s~ tmpdir:s layer-dir:s','container'),
 'disk_settings':('ResourceDiskSettings','/disk/settings','auto-media-interface:s auto-media-sharing:b auto-smb-sharing:b default-mount-point-template:s',None),
 'ip_dns':('ResourceDns','/ip/dns','servers:s+ allow-remote-requests:b cache-size:i max-concurrent-queries:i max-concurrent-tcp-sessions:i max-udp-packet-size:i verify-doh-cert:b vrf:s',None),
 'interface_detect_internet':('ResourceInterfaceDetectInternet','/interface/detect-internet','detect-interface-list:s internet-interface-list:s lan-interface-list:s wan-interface-list:s',None),
 'interface_l2tp_server':('ResourceInterfaceL2tpServer','/interface/l2tp-server/server','enabled:b default-profile:s max-mtu:i max-mru:i authentication:s+ allow-fast-path:b one-session-per-host:b',None),
 'interface_sstp_server':('ResourceInterfaceSSTPServer','/interface/sstp-server/server','enabled:b default-profile:s certificate:s max-mtu:i max-mru:i authentication:s+ verify-client-certificate:b pfs:s',None),
 'interface_wireless_cap':('ResourceInterfaceWirelessCap','/interface/wireless/cap','enabled:b bridge:s caps-man-addresses:s+ discovery-interfaces:s+ interfaces:s+ lock-to-caps-man:b','wireless'),
 'ip_cloud':('ResourceIpCloud','/ip/cloud','ddns-enabled:s update-time:b dns-name:s* public-address:s* status:s*',None),
 'ip_cloud_advanced':('ResourceIpCloudAdvanced','/ip/cloud/advanced','use-local-address:b',None),
 'ip_smb':('ResourceIpSMB','/ip/smb','enabled:s interfaces:s+ domain:s comment:s status:s*',None),
 'ip_ssh_server':('ResourceIpSSHServer','/ip/ssh','strong-crypto:b forwarding-enabled:s',None),
 'ppp_aaa':('ResourcePppAaa','/ppp/aaa','accounting:b enable-ipv6-accounting:b use-radius:b use-circuit-id-in-nas-port-id:b',None),
 'snmp':('ResourceSNMP','/snmp','enabled:b contact:s location:s src-address:s trap-version:i vrf:s',None),
 'system_clock':('ResourceSystemClock','/system/clock','time-zone-name:s time-zone-autodetect:b dst-active:b* gmt-offset:s*',None),
 'system_led_settings':('ResourceSystemLedSettings','/system/leds/settings','all-leds-off:s',None),
 'system_user_settings':('ResourceSystemUserSettings','/user/settings','minimum-password-length:i minimum-categories:i',None),
 'tool_bandwidth_server':('ResourceToolBandwidthServer','/tool/bandwidth-server','enabled:b authenticate:b allocate-udp-ports-from:i max-sessions:i',None),
 'tool_email':('ResourceToolEmail','/tool/e-mail','server:s port:i user:s password:s~ from:s tls:s vrf:s last-status:s*',None),
 'ip_upnp':('ResourceUPNPSettings','/ip/upnp','enabled:b allow-disable-external-interface:b show-dummy-rule:b',None),
 'system_user_aaa':('ResourceUserAaa','/user/aaa','accounting:b default-group:s exclude-groups:s+ use-radius:b',None),
 'user_manager_advanced':('ResourceUserManagerAdvanced','/user-manager/advanced','paypal-allow:b paypal-use-sandbox:b paypal-user:s~ paypal-password:s~ paypal-signature:s~','user-manager'),
 'user_manager_database':('ResourceUserManagerDatabase','/user-manager/database','db-path:s','user-manager'),
 'user_manager_settings':('ResourceUserManagerSettings','/user-manager','enabled:b authentication-port:i accounting-port:i certificate:s use-profiles:b require-message-auth:s','user-manager'),
 'wifi_cap':('ResourceWifiCap','/interface/wifi/cap','enabled:b caps-man-addresses:s+ discovery-interfaces:s+ lock-to-caps-man:b slaves-static:b',None),
 'wifi_capsman':('ResourceWifiCapsman','/interface/wifi/capsman','enabled:b interfaces:s+ require-peer-certificate:b certificate:s ca-certificate:s upgrade-policy:s',None),
}
PATHS={k:v[1] for k,v in SETTINGS.items()}
CHOICES={
 ('interface_sstp_server','pfs'):['no','yes','required'],
 ('ip_cloud','ddns_enabled'):['auto','yes'],
 ('ip_smb','enabled'):['auto','yes','no'],
 ('ip_ssh_server','forwarding_enabled'):['no','local','remote','both'],
 ('system_led_settings','all_leds_off'):['never','after-1h','immediate'],
 ('tool_email','tls'):['no','yes','starttls'],
 ('wifi_capsman','upgrade_policy'):['none','suggest-same-version','require-same-version'],
 ('capsman_manager','upgrade_policy'):['none','suggest-same-version','require-same-version'],
 ('user_manager_settings','require_message_auth'):['no','yes-access-request'],
}
BOUNDS={
 ('ip_dns','cache_size'):(64,4294967295),
 ('ip_dns','max_concurrent_queries'):(1,65535),
 ('ip_dns','max_concurrent_tcp_sessions'):(1,65535),
 ('ip_dns','max_udp_packet_size'):(512,65535),
 ('interface_l2tp_server','max_mtu'):(128,65535),
 ('interface_l2tp_server','max_mru'):(128,65535),
 ('interface_sstp_server','max_mtu'):(128,65535),
 ('interface_sstp_server','max_mru'):(128,65535),
 ('snmp','trap_version'):(1,3),
 ('system_user_settings','minimum_password_length'):(0,255),
 ('system_user_settings','minimum_categories'):(0,4),
 ('tool_bandwidth_server','allocate_udp_ports_from'):(1,65535),
 ('tool_bandwidth_server','max_sessions'):(1,65535),
 ('tool_email','port'):(1,65535),
 ('user_manager_settings','authentication_port'):(1,65535),
 ('user_manager_settings','accounting_port'):(1,65535),
}
def policies():
 result=[]
 for name,(constructor,path,tokens,package) in SETTINGS.items():
  fields=[]
  for token in ['.id:s*']+tokens.split():
   wire,kind=token.split(':');field='id' if wire=='.id' else wire.replace('-','_').replace('.','_')
   typ={'s':'string','i':'integer','b':'boolean'}[kind[0]];mode='computed' if '*' in kind else 'required' if '!' in kind else 'computed_optional'
   f={'name':field,'wire':wire,'type':typ,'mode':mode,'force_new':False,'sensitive':'~' in kind,'codec':'csv-string-set' if '+' in kind else {'string':'wire-string','integer':'decimal','boolean':'yes/no'}[typ],'enum_kind':'none','default':'server-owned' if mode=='computed_optional' else None,'validators':'maintained additional settings scalar validation','provenance':'explicit reviewed additional settings subset; pinned static SDK declarations plus separately verified publication; no SDK migration claim'}
   if '~' in kind:f['preserve_secret_on_omission']=True
   if (name,field) in CHOICES:f['choices']=CHOICES[name,field]
   if (name,field) in BOUNDS:f['minimum'],f['maximum']=BOUNDS[name,field]
   fields.append(f)
  p={'revision':'remaining-resources-settings-curated-v1','reference_constructor':constructor,'resource_name':name,'wire_path':path,'lifecycle':'singleton','schema_version':0,'migration_compatibility':'not claimed','reference_sha':REFERENCE,'attributes':fields,'required':[f['name'] for f in fields if f['mode']=='required'],'optional_computed':[f['name'] for f in fields if f['mode']=='computed_optional'],'computed':[f['name'] for f in fields if f['mode']=='computed'],'booleans':[f['name'] for f in fields if f['type']=='boolean']}
  if package:p['required_package']=package
  result.append(p)
 return result
if __name__=='__main__':
 (ROOT/'internal/catalog/additional-settings.json').write_text(json.dumps(policies(),indent=2)+'\n')
