# Batch A — 107 integrations, final delivery gates in progress

The fixed roster remains **107 distinct constructors** (88 collections and 19
singletons); **125 public constructors** including the eighteen preserved baseline
resources. Batch B's other 107 constructors remain unexposed. Aliases do not count
twice. Final milestone completion is NOT claimed until clean-source consolidated
live evidence, immutable maintenance replay/no-op and hosted verification complete.

## Constructor-level implementation tracker

This is the sole current tracker. Frozen ledger statuses remain historical planning
evidence, never an exposure selector. `pass` below means observed focused tests;
live results so far bind dirty source, not a maintenance candidate or release.

| Constructor / canonical resource | Reviewed contract | Lifecycle | Official generation | Registration | Mock/race | Live |
| --- | --- | --- | --- | --- | --- | --- |
| `ResourceCertificateScepServer` / `routeros_system_certificate_scep_server` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceDhcpClient` / `routeros_ip_dhcp_client` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceDhcpRelay` / `routeros_ip_dhcp_relay` | explicit subset | CRUD, import | both pinned | public | pass | both base lanes |
| `ResourceDhcpServerConfig` / `routeros_ip_dhcp_server_config` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceDhcpServerOptionMatcher` / `routeros_ip_dhcp_server_option_matcher` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceDhcpServerOptionSets` / `routeros_ip_dhcp_server_option_set` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceDnsAdlist` / `routeros_ip_dns_adlist` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceDnsRecord` / `routeros_ip_dns_record` | explicit subset | CRUD, import | both pinned | public | pass | both base lanes |
| `ResourceIPConnectionTracking` / `routeros_ip_firewall_connection_tracking` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceIPVrf` / `routeros_ip_vrf` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIPv6Address` / `routeros_ipv6_address` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIPv6DhcpClient` / `routeros_ipv6_dhcp_client` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIPv6DhcpClientOption` / `routeros_ipv6_dhcp_client_option` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIPv6FirewallAddrList` / `routeros_ipv6_firewall_addr_list` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIPv6FirewallFilter` / `routeros_ipv6_firewall_filter` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIPv6NeighborDiscovery` / `routeros_ipv6_neighbor_discovery` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIPv6Route` / `routeros_ipv6_route` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterface6to4` / `routeros_interface_6to4` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceBonding` / `routeros_interface_bonding` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceBridgeSettings` / `routeros_interface_bridge_settings` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceInterfaceBridgeVlan` / `routeros_interface_bridge_vlan` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceDot1xClient` / `routeros_interface_dot1x_client` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceDot1xServer` / `routeros_interface_dot1x_server` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceGre6` / `routeros_interface_gre6` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceL2tpClient` / `routeros_interface_l2tp_client` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceMacVlan` / `routeros_interface_macvlan` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceOpenVPNServer` / `routeros_interface_ovpn_server` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfacePPPoEClient` / `routeros_interface_pppoe_client` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfacePppoeServer` / `routeros_interface_pppoe_server` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceSSTPClient` / `routeros_interface_sstp_client` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceVeth` / `routeros_interface_veth` | explicit subset | CRUD, import | both pinned | public | pass | both container lanes; unavailable on base |
| `ResourceInterfaceVxlan` / `routeros_interface_vxlan` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceVxlanVteps` / `routeros_interface_vxlan_vteps` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceWireguard` / `routeros_interface_wireguard` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceInterfaceWireguardPeer` / `routeros_interface_wireguard_peer` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpDnsForwarders` / `routeros_ip_dns_forwarders` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpFirewallLayer7Protocol` / `routeros_ip_firewall_layer7_protocol` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpHotspot` / `routeros_ip_hotspot` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpHotspotIpBinding` / `routeros_ip_hotspot_ip_binding` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpHotspotProfile` / `routeros_ip_hotspot_profile` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpHotspotUser` / `routeros_ip_hotspot_user` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpHotspotUserProfile` / `routeros_ip_hotspot_user_profile` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpHotspotWalledGarden` / `routeros_ip_hotspot_walled_garden` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpHotspotWalledGardenIp` / `routeros_ip_hotspot_walled_garden_ip` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpIpsecIdentity` / `routeros_ip_ipsec_identity` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpIpsecModeConfig` / `routeros_ip_ipsec_mode_config` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpIpsecPeer` / `routeros_ip_ipsec_peer` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpIpsecPolicy` / `routeros_ip_ipsec_policy` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpIpsecPolicyGroup` / `routeros_ip_ipsec_policy_group` | explicit subset | replacement-only, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpIpsecProfile` / `routeros_ip_ipsec_profile` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpIpsecProposal` / `routeros_ip_ipsec_proposal` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpIpsecSettings` / `routeros_ip_ipsec_settings` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceIpNeighborDiscoverySettings` / `routeros_ip_neighbor_discovery_settings` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceIpSettings` / `routeros_ip_settings` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceIpTFTP` / `routeros_ip_tftp` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpTFTPSettings` / `routeros_ip_tftp_settings` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceIpTrafficFlow` / `routeros_ip_traffic_flow` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceIpTrafficFlowIpfix` / `routeros_ip_traffic_flow_ipfix` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceIpTrafficFlowTarget` / `routeros_ip_traffic_flow_target` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpv6DhcpServer` / `routeros_ipv6_dhcp_server` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpv6DhcpServerOption` / `routeros_ipv6_dhcp_server_option` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpv6DhcpServerOptionSets` / `routeros_ipv6_dhcp_server_option_sets` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpv6NdPrefix` / `routeros_ipv6_nd_prefix` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpv6Pool` / `routeros_ipv6_pool` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceIpv6Settings` / `routeros_ipv6_settings` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceNatPmpInterfaces` / `routeros_ip_nat_pmp_interfaces` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceNatPmpSettings` / `routeros_ip_nat_pmp` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceOpenVPNClient` / `routeros_interface_ovpn_client` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourcePPPProfile` / `routeros_ppp_profile` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourcePPPSecret` / `routeros_ppp_secret` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceQueueSimple` / `routeros_queue_simple` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceQueueTree` / `routeros_queue_tree` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceQueueType` / `routeros_queue_type` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRadius` / `routeros_radius` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRadiusIncoming` / `routeros_radius_incoming` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceRoutingBfdConfiguration` / `routeros_routing_bfd_configuration` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingBgpConnection` / `routeros_routing_bgp_connection` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingBgpEvpn` / `routeros_routing_bgp_evpn` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingBgpInstance` / `routeros_routing_bgp_instance` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingBgpTemplate` / `routeros_routing_bgp_template` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingBgpVpn` / `routeros_routing_bgp_vpn` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingFilterRule` / `routeros_routing_filter_rule` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingId` / `routeros_routing_id` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingIgmpProxyInterface` / `routeros_routing_igmp_proxy_interface` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingOspfArea` / `routeros_routing_ospf_area` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingOspfAreaRange` / `routeros_routing_ospf_area_range` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingOspfInstance` / `routeros_routing_ospf_instance` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingOspfInterfaceTemplate` / `routeros_routing_ospf_interface_template` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceRoutingRule` / `routeros_routing_rule` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceSNMPCommunity` / `routeros_snmp_community` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceSystemIdentity` / `routeros_system_identity` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceSystemLogging` / `routeros_system_logging` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceSystemNote` / `routeros_system_note` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceSystemNtpClient` / `routeros_system_ntp_client` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceSystemNtpServer` / `routeros_system_ntp_server` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceSystemScheduler` / `routeros_system_scheduler` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceToolGraphingInterface` / `routeros_tool_graphing_interface` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceToolGraphingQueue` / `routeros_tool_graphing_queue` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceToolGraphingResource` / `routeros_tool_graphing_resource` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceToolMacServer` / `routeros_tool_mac_server` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceToolMacServerPing` / `routeros_tool_mac_server_ping` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceToolMacServerWinBox` / `routeros_tool_mac_server_winbox` | explicit subset | GET object, POST /set, unmanage | both pinned | public | pass | both base lanes |
| `ResourceToolNetwatch` / `routeros_tool_netwatch` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceUPNPInterfaces` / `routeros_ip_upnp_interfaces` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceUser` / `routeros_system_user` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceUserGroup` / `routeros_system_user_group` | explicit subset | CRUD, import | both pinned | public | pass | both base + container lanes |
| `ResourceUserSshKeys` / `routeros_system_user_sshkeys` | explicit subset | replacement-only, import | both pinned | public | pass | both base + container lanes |

## Reviewed runtime scope and limitations

- Contracts: `internal/catalog/{collections,singletons,batch-a-collections}.json`.
  The maintained policy builders explicitly enumerate fields; unlisted fields,
  actions, aliases and SDK upgraders remain unsupported. SDK migration and parity
  are not claimed. Official OpenAPI v0.3.0 and Framework v0.4.1 generators emit
  all schemas/models. `tools/schema/bindings.py` emits only function references.
- Collections share bounded CRUD, ID import, ownership guards, omission, explicit
  clearing, atomic readback, deletion recreation and retained allocated IDs after
  create refresh failure. Scalar CSV sets preserve equivalent order; durations
  preserve equivalent spelling. MAC/VLAN/IP formats and selected bounds/enums
  are reviewed. Automatic MTU/hop-limit values remain null rather than becoming
  fabricated numbers. User permission readback handles explicit negative flags.
- SSH keys: required sensitive user-owned public key, replacement-only mutations,
  SHA256 fingerprint ownership verification including RouterOS padding, omitted-key
  state preservation. Import cannot recover key material; supplying a key after
  fresh import requires replacement. Administrator/authentication-user keys cannot
  be managed. Unsupported comment is omitted; key-owner metadata is sensitive.
- VETH: actual extra publication is separately hash-bound; base paths are never
  fabricated. Runtime requires enabled matching-version `container`. Pinned
  official package recipes provision owned disposable guests via password SFTP,
  fixed reboot and signature-verified RouterOS installation; no user key changes
  or container execution. Both pinned container lanes passed.
- Nineteen singletons: fixed GET-object/POST-`/set`, path-derived identity/import,
  destroy-unmanage only, atomic malformed reads and failure recovery. Destroy
  never DELETEs, resets or writes global settings. NTP lists/choices/bounds have
  explicit validation and canonical readback.
- Disabled/disconnected/unattached live objects preserve connectivity. The owned
  CA prerequisite only serves disabled SCEP configuration; scheduler uses a
  disabled harmless body, Netwatch imports with hidden callbacks fail closed.
  No packet-processing, arbitrary script execution or credential rotation is
  certified. OSPF passive and ignored ND/IPsec-group/SSH comments are unexposed.
  The IPsec policy-group has only a meaningful replacement-owned name because
  RouterOS does not support its purported comment field.
- All eighteen accepted descriptors and generated files were independently
  compared with `1897fb32c03c43fb48f6ab3020cc3baf2e2b57b2`: unchanged.
  Historical matrices, receipts, ledger and examples are preserved.

## Historical foundation evidence (superseded implementation status)

The sections below record the earlier groundwork checkpoint. Their statements
about unregistered singleton support and pending implementation describe that
historical checkpoint, not the current table above.

## Frozen ledger and immutable observations

[`schemas/batch-a-ledger.json`](../../schemas/batch-a-ledger.json) records all 107
IDs, explicit canonical-name proposals, deferred aliases, source locations,
static field candidates/unresolved declarations, triage risks, and path/method
observations for **7.24.5/base, 7.24.5/extra, 7.25beta5/base and 7.25beta5/extra**.
Its 107 complement IDs keep Batch B fixed. Pending statuses and empty evidence
are deliberate: inventory and observations do not authorize exposure.

Source baseline: `1897fb32c03c43fb48f6ab3020cc3baf2e2b57b2`; SDK reference:
`0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb`; published restraml input:
`adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a`. Every schema input and the exact
historical baseline matrix bytes are hash-bound. The archived matrix is distinct
from the current capability matrix, so later implementation status changes do
not shrink the denominator or invalidate the partition test in shallow CI.

```sh
python3 tools/contracts/batch_a.py --upstream-dir ../restraml
python3 -m unittest discover -s tools/contracts -p 'test_batch_a.py'
```

The builder reads committed blobs only, never upstream callbacks/scripts. It
rejects changed baseline bytes and refuses to overwrite differing ledger content.
Canonical names are **proposals**, not registrations or automatic aliases.

Observed shapes in **each base lane**: 86 full-CRUD collection candidates,
19 singleton candidates, one absent collection and one partial collection.
In **each extra lane**: 87 full-CRUD collection candidates, 19 singleton
candidates and one partial collection. Structural CRUD is not semantic approval.

Two cases specifically require maintained behavior rather than blanket CRUD:

- **`ResourceInterfaceVeth`:** absent from both pinned base publications, present
  with structural CRUD in both extra publications. Resolve published extra input,
  package/topology provisioning, runtime capability and lane-aware generation
  without fabricating base paths or silently weakening exact resource-set gates.
- **`ResourceUserSshKeys`:** no item PATCH in any of the four publications. The
  pinned `resource_system_user_sshkeys.go` declares required sensitive,
  replacement-owned `key`, plus user/comment and computed fingerprint metadata.
  Resolve replacement-only mutation policy and missing-key readback preservation;
  never PATCH an unsupported path, manage the acceptance admin's keys, or leak key
  material through errors/evidence.

Other unresolved rows stay inside Batch A. OSPF interface-template construction,
sets/nesting, secrets, ownership, async/import behavior and canonical wire values
still require explicit field/action contracts and tests, not a blanket approval.

## Maintained singleton lifecycle foundation

`internal/provider/singleton_resource.go` is **unregistered support**, not nineteen
implemented resources. Its explicit lifecycle follows static reference
`resource_actions.go:311–355` and `resource_actions_default_system.go`:

- Read the singleton object through GET at its fixed policy path.
- Create-as-update and update POST the fixed `/set` action. Never PUT a collection,
  PATCH an item, transmit `.id`, or write computed fields.
- Identity derives from the path (`/system/identity` → `system.identity`). Import
  rejects item IDs and other singleton identities before any network operation.
- Destroy only relinquishes Terraform management, retaining device settings and
  warning the user. It requires no connection and never resets/deletes settings.
- Preflight reads precede writes. A failed create write/read retains the known
  baseline and deterministic identity for recovery. Malformed/unauthorized/404
  reads preserve state and report errors rather than silently removing a global
  settings resource.
- Primitive string/boolean/int64 payload and atomic readback are tested. Omitted
  writable fields stay omitted, explicit false becomes `no`, empty note clears,
  multiline notes remain supported, and unknown mutations/control characters are
  rejected. Collection replacement/default/conditional/array policies are rejected
  until their singleton equivalents have reviewed implementations.

Test-only schemas and a test-only provider exercise the engine; they do **not**
replace the two required official generators. Before exposing each of the nineteen
singletons, add its reviewed validators/defaults/field semantics, adapter and
exact generated-file/spec gates, official generated schema/model binding, and
applicable live evidence. Runtime registration remains unchanged meanwhile.

## Amortized immutable-input adaptation

`SchemaSnapshot` in `tools/schema/normalize/normalize.py` parses and validates the
same immutable input once per CLI invocation and observes inventory once. Every
resource still independently passes the existing exact lifecycle/field/codec/
replacement/ownership-policy checks. Different input hashes cannot reuse a cache;
report classifications are copied and cannot mutate sibling observations.
Secondary per-resource reports omit an unused full inventory copy; the final
bundle still contains the complete observed inventory and exact reviewed set.

The cached CLI produced byte-identical current normalized OpenAPI, generator
configuration, wire descriptors and upstream input compared with the existing
artifacts. No generated model or public schema changed. A **single local
18-policy fixture sample** measured 0.098s without reuse versus 0.013s with reuse
(7.59×); this saves only 0.085s on that fixture and is **not** a milestone timing
estimate or a substitute for larger batching and actual implementation work.

## Targeted checks actually observed

- `go test ./internal/provider -run '^TestSingleton' -count=1 -race`: passed,
  including a real Terraform CLI mock create/update/import, empty plans, drift
  repair, destroy-retains-settings, malformed refresh and recovery cases.
- Normalizer tests: **26 passed** (including five new snapshot-isolation/gate tests).
- Contract tooling tests: **31 passed** (including five planning-ledger tests).
- Cached CLI adaptation bundle verification and four current artifact byte
  comparisons: passed.
- `go vet ./internal/provider` and `git diff --check`: passed.

No new CHR guests, credentials, hosted runs or pushes were created for this
foundation work. No full 125-resource regression or final maintenance replay has
occurred. Existing eighteen-resource historical evidence is unchanged. These foundation
check results predate the public integrations recorded above.
