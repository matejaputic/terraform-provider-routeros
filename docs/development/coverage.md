# Resource coverage

Runtime revision `framework-rest-preview-v10-batch-a`; **125 reviewed resource constructors**, zero data sources. The fixed 107 Batch A integrations are generated and registered, with focused mock/race and both pinned live lanes passing; VETH additionally passed both matching container-package lanes. Final clean-source consolidated verification/maintenance/hosted delivery gates remain in progress. See the sole [constructor tracker and subset limitations](batch-a.md). The table below retains the original twenty-resource contract details.

| Resource | Implementation | Terraform mock CRUD/import | Live CHR CRUD/import |
| --- | --- | --- | --- |
| `routeros_ip_address` | Maintained IP-specific lifecycle | Pass | Pass |
| `routeros_interface_bridge` | Reviewed collection lifecycle | Pass | Pass |
| `routeros_interface_bridge_port` | Reviewed collection lifecycle | Pass | Pass |
| `routeros_interface_vlan` | Reviewed collection lifecycle | Pass | Pass |
| `routeros_interface_list` | Reviewed collection lifecycle | Pass | Pass |
| `routeros_interface_list_member` | Reviewed collection lifecycle | Pass | Pass |
| `routeros_ip_pool` | Reviewed collection lifecycle + CSV ranges | Pass | Pass |
| `routeros_ip_dhcp_server_network` | Reviewed DHCP network + typed DNS CSV | Pass | Pass |
| `routeros_ip_dhcp_server` | Reviewed named collection lifecycle | Pass | Pass |
| `routeros_ip_dhcp_server_lease` | Static-only collection + MAC equivalence | Pass | Pass |
| `routeros_ip_route` | Explicit static-only collection | Pass | Pass |
| `routeros_ip_firewall_addr_list` | Permanent IPv4 entries + numeric address equivalence | Pass | Pass |
| `routeros_ip_firewall_filter` | Maintained static rules + explicit chain-relative ordering | Pass | Pass |
| `routeros_ip_firewall_nat` | Reviewed NAT targets/actions + explicit ordering | Pass | Pass |
| `routeros_ip_firewall_mangle` | Connection/packet marking + explicit ordering | Pass | Pass |
| `routeros_ip_firewall_raw` | Reviewed notrack/drop/basic actions + explicit ordering | Pass | Pass |
| `routeros_ip_dhcp_client_option` | Reviewed option lifecycle + default-object guard | Pass | Pass (7.24.5 / 7.25beta5) |
| `routeros_ip_dhcp_server_option` | Reviewed option lifecycle + raw readback/force omission | Pass | Pass (7.24.5 / 7.25beta5) |
| `routeros_ip_dhcp_relay` | Disabled-capable relay subset + IPv4 destinations | Pass | Pass (both base lanes, dirty-source focused suite) |
| `routeros_ip_dns_record` | Named A/AAAA subset + replacement type | Pass | Pass (both base lanes, dirty-source focused suite) |

All 107 Batch A constructors are integrated; final milestone delivery is not yet claimed. The original [two-resource focused evidence](../../schemas/batch-a-integration-validation.json) and following clean-source statements retain their historical scope.

The new option wave and all original regressions passed from clean revision `06d10a6` on both local pinned lanes; [wave evidence](../../schemas/dhcp-options-wave-validation.json) records exact bindings. The historical hosted evidence below remains specific to its original sixteen-resource revision.

Original baseline live scope: disposable **RouterOS 7.24.5 x86_64/base CHR** on OrbStack Docker/QEMU TCG. At clean source revision `93d3527`, the complete five-suite/sixteen-resource configuration acceptance also passed on pinned **7.25beta5 x86_64/base**, locally and on isolated GitHub-hosted QEMU/TCG; the baseline passed again in both environments. [Hosted evidence](../../schemas/hosted-chr-validation.json) binds exact targets, enabled packages, image/recipe hashes and run URLs. This is source-revision lifecycle evidence, not proof for a newly generated maintenance candidate or released binary. No other version, architecture, extra package lane, native API transport or SDK-state migration is newly certified. Each resource exposes a reviewed field subset, **not every reference-provider field**. Read the individual schema docs; unknown attributes produce Terraform diagnostics rather than silent omissions.

## Maintained collection behavior

`internal/catalog/collections.json` preserves nineteen accepted collection subsets; `singletons.json` and `batch-a-collections.json` explicitly authorize the remaining reviewed subsets. This file is embedded in the runtime and bound into discovery producer fingerprints; schema inventory never authorizes registration. The snapshot adapter supports the reviewed string/bool/int64/string-list types, aliases, decimal/boolean/CSV codecs and name replacement. Official OpenAPI and Framework generators produce 125 models/schemas. Independent gates require the complete exact resource/field/type/mode set before staged files replace last-good output. Name replacement modifiers and all lifecycle methods remain maintained code.

CRUD uses PUT create, filtered collection GET by `.id`, PATCH update and DELETE item. Import accepts only an internal `*HEX` ID. Actual collection absence/typed 404 is distinguished from auth, malformed, partial and transport failures. Refresh commits a temporary model only after successful validation. Writes contain configured writable fields only, never computed state or IDs. Failed create-read preserves a known ID/partial state. Built-in or dynamic records cannot be imported/managed/deleted through this ordinary collection lifecycle. Changing a `name` replaces the object, matching the reviewed reference behavior, rather than silently renaming dependencies.

Reviewed omission semantics: empty comment and empty list include/exclude serialize back as empty strings; absent pool `next-pool` means `none`. These explicit exceptions are not a generic missing-field fallback. Bridge `pvid` is hidden by RouterOS when VLAN filtering is disabled, even with `.proplist`; configuring it requires explicit `vlan_filtering = true`. Unconfigured/hidden PVID is null; genuine missing configured fields otherwise fail read.

Pool `ranges` is a required nonempty list of IPv4 addresses or ascending `start-end` ranges. IPv6, CIDR, overlapping/duplicate/reversed ranges and null/unknown elements at apply are rejected. CSV read/write is typed; equivalent numeric ranges retain configuration order/single-IP spelling to avoid server reserialization drift. Different bounds remain visible. Import uses the server representation, so use canonical range ordering when adopting an existing pool. VLAN IDs/PVIDs require 1–4094, MTU 68–65535; reviewed bridge protocol/edge modes are validated before writes.

## DHCP and static routing

These four resources expose reviewed subsets. DHCP networks require canonical IPv4 CIDR; gateway and DNS servers are IPv4, DNS is a typed list (empty clears it), netmask is 0–32. `dns_none = true` and a nonempty DNS list is rejected: live RouterOS clears the servers regardless of the desired configuration. Absent DNS/domain read as empty, absent dns-none as false. Server address_pool is required rather than guessing a pool; time formats, lease scripts, options and secret/sensitive fields remain deferred. Reviewed omitted server add-arp means false, conflict-detection means true, and static-lease block-access means false. MAC addresses must be 48-bit Ethernet addresses; equivalent configured case/separator spelling is preserved, but import adopts remote canonical spelling. Only static leases are managed; dynamic leases are rejected rather than converted.

Routes require explicit canonical IPv4 `dst_address` and `gateway`; no implicit default route. Distance is 1–255, scopes 0–255. Only records with explicit static=true and no dynamic=true are manageable. Missing static flag fails closed; absent active/dynamic flags mean false. Blackhole/presence-sensitive flags, dynamic routes and advanced routing-table lifecycles are not exposed. Name replacements involving dependent objects may require Terraform `create_before_destroy` on named parents; no automatic dependency renaming is promised.

`TestAccDHCPRoutingCHR` creates these four resources with an isolated bridge/pool/IP topology and verifies CRUD, imports, empty plans/applies, DNS list changes and explicit clearing, domain/comment clearing, add-arp/block-access updates, distance updates and externally deleted static-route recreation. DHCP server stays disabled; no actual DHCP client exchange or packet-forwarding performance is certified. Only documentation prefixes 192.0.2.0/24 and 198.51.100.0/24 are used, never management/default routes. Destroy checks filter exact test objects and the whole VM is removed afterwards.

## Firewall address lists and filter ordering

Address-list resource name matches the reviewed reference: `routeros_ip_firewall_addr_list`. Permanent IPv4 address/CIDR/ascending-range entries only; hostname resolution, IPv6, finite lifetimes and dynamic records are not implemented. Canonical CIDR is required; numeric bounds preserve equivalent configured range or /32 spelling when RouterOS rewrites it. Import adopts the remote canonical address. Live range import checks the lifecycle but intentionally ignores address-spelling comparison; use the server's CIDR spelling when adopting such entries.

Filter fields are a reviewed subset: chain, action, source/destination address-list names, comment, disabled, log, dynamic, invalid and place_before. Supported actions: accept/drop/log/passthrough/return; jump/reject/fasttrack/add-to-list and their dependent settings are deferred. Empty address-list names clear those matches. `place_before` accepts a same-chain static rule's internal *HEX ID; explicitly `""` means chain end. Omitted means unmanaged placement (append on create); computed readback reports the next rule in the same chain, not a global numeric index. Create/update omit placement from ordinary PUT/PATCH and use the reviewed `/ip/firewall/filter/move` command only after validating subject/anchor IDs and static chain membership. Post-move read verifies the requested position. Refresh derives placement from collection order so external moves cause a real plan and repair. Own rule movement does not change the relative order of other rules. Import reports actual next same-chain ID. Self/missing/cross-chain/dynamic anchors and duplicate/malformed collection rows fail closed. A chain containing dynamic rules is deliberately unsupported. REST is not transactional: concurrent external moves or competing desired positions can still fail diagnostics; no global router ordering lock is promised.

Live `TestAccFirewallCHR` uses only a new **unreferenced** `tf-coverage-filter` chain with disabled rules and a `tf-coverage-firewall` address list. It tests range-to-CIDR consistency, CRUD/import, comments, logging, address updates, placement before an anchor, moving to chain end and back, external reordering repair, and empty plans. No input/forward/output management-access rule is mutated, and no packet-processing correctness/performance is certified. Mock tests additionally assert placement never leaks into regular mutation payloads; unit tests cover unsafe anchor/snapshot rejection without writes.

## NAT, mangle and raw

These three rule families use separate fixed menu/move bindings; a catalog path cannot authorize arbitrary actions. All share the static-chain ordering/import/drift behavior above. Public fields remain curated subsets, not full reference parity. NAT supports accept/log/passthrough/return/masquerade/src-nat/dst-nat; src-nat/dst-nat require `to_addresses` as an IPv4 address or ascending range, not CIDR/DNS/IPv6. No to_ports/protocol/port matcher or redirect/netmap/same/endpoint-independent-NAT support is exposed yet. Nonempty configured targets for unrelated actions are rejected.

Mangle supports accept/log/passthrough/return/mark-connection/mark-packet. Mark actions require the corresponding nonempty new_connection_mark/new_packet_mark. Configured marks must match the action; passthrough is only authorized for these two mark actions. Routing marks, DSCP/MSS/TTL changes, sniffing and their required dependencies remain deferred. Raw supports accept/drop/log/notrack/passthrough/return. Jump and add-to-list actions remain deferred across these resources.

Action applicability validates configuration intent separately from readback: RouterOS can retain inactive fields after an action change. Omitted optional+computed fields adopt server state instead of implicitly clearing it. Missing inactive target/mark strings read as empty; missing configured required targets still fail validation. Mock and live tests cover all three families' CRUD/import, before/end/back ordering, external reorder repair, chain relocation, logging/match/comment updates and empty plans. Live NAT additionally exercises src-nat → dst-nat → masquerade → src-nat; mangle exercises connection → packet → connection marking and passthrough false/true; raw exercises notrack/drop. All rules stay disabled in unreferenced documentation/test chains. This is configuration acceptance, **not traffic translation/marking/notrack correctness or performance certification**.

## Live evidence and isolation

`TestAccCollectionsCHR` creates a dependent bridge → VLAN / physical test port → interface list member topology plus an IP pool. It verifies create/read, numeric/bool/list state, comment and meaningful integer/bool/range updates, import for all six, empty subsequent plans/applies, out-of-band pool deletion/recreation, name replacement and final destroy. Existing IP acceptance still passes.

The harness adds an isolated second virtio NIC on a QEMU-only hub with **no host/network backend**, named `tf-port`. The management NIC remains untouched; existing `tf-test` is a disconnected bridge for IP tests. No KVM, privileged containers or host networking. Base uses only loopback REST/serial ports; explicitly requested container lanes briefly use a loopback password-SFTP provisioning port, then disable SSH. All test bridge/VLAN/list/pool names use `tf-coverage-`; cleanup verifies owned records and then destroys the entire guest/container, credentials and mutable disk even after test failures.

Next work is the consolidated Batch A delivery gate, not another inventory or small resource wave. Release infrastructure (generated-candidate acceptance and durable tested/published receipts) remains separate unfinished work. Hardware/Batch B, unlisted fields/actions, traffic behavior and SDK-state migration remain unexposed or uncertified. Singletons and SSH keys use their reviewed special lifecycles, never invented blanket CRUD.
