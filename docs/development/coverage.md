# Resource coverage and validation scope

The current provider exposes **152 canonical resource constructors**: **106 collections**, **46 settings/singletons**, and **zero data sources**. The [resource index](../resources/index.md) lists every exposed resource and links its exact reviewed schema. The pinned static reference inventory contains 256 resource names / 232 constructors; aliases do not count twice, and **80 constructors remain unimplemented**. Neither inventory nor a planning ledger authorizes exposure.

## Reviewed registration

| Source | Resources | Runtime |
| --- | ---: | --- |
| IP-address policy and `NewIPAddressResource` | 1 | IP-specific collection lifecycle |
| `internal/catalog/collections.json` | 19 | Reviewed networking, DHCP, firewall, relay and DNS collection lifecycles |
| `internal/catalog/extended-collections.json` | 86 | Reviewed collection lifecycles, including two replacement-only resources |
| `internal/catalog/singletons.json` | 19 | Existing settings: GET object / POST fixed `/set` / unmanage on destroy |
| `internal/catalog/additional-settings.json` | 27 | Additional reviewed settings using the same unmanage-only lifecycle |
| **Total** | **152** | **106 collections / 46 settings** |

Official OpenAPI v0.3.0 and Framework v0.4.1 generators emit all 152 schemas/models into `internal/generated/`. The binding scripts emit references to official schema functions, not custom schemas. Maintained runtime stays outside that directory. Independent normalized-schema, code-spec and exact-file-set gates remain fail-closed. Runtime/provenance metadata retains revision `framework-rest-preview-v10-reviewed-resources`; the actual constructor set is determined by registration and descriptors, not that historical label.

## Recorded evidence, not inferred certification

| Set | Offline/mock | Live configuration scope | Limits |
| --- | --- | --- | --- |
| Preserved 125 constructors | Full race/mock and official-generation gates | Clean source `c61b371`: both pinned base and matching container lanes; VETH explicitly unavailable on base. Inspected hosted bindings at `48d6b43`. | Historical source/configuration evidence, not current released-binary or maintenance-candidate certification |
| 27 additional settings | All schema/atomic-failure/secret checks and Terraform mock lifecycles; recorded full provider coverage 80.8% | Focused **dirty-source** suites on both pinned versions: 19 available on base, 26 with matching container/wireless/user-manager packages | Physical LED controls unavailable on CHR; no recorded consolidated clean-source/hosted or expanded maintenance replay for the full 152-resource set |

Exact records: [configuration validation](../../schemas/resource-configuration-validation.json), [hosted bindings](../../schemas/hosted-configuration-validation.json), and [additional settings validation](../../schemas/additional-settings-validation.json). Renamed path/selector labels in historical evidence do not change the historical digests or certify the renamed/current checkout.

`tools/chr/chr.py test` and the four hosted CHR jobs currently select the preserved 125-constructor acceptance suites. **They do not select `TestAccAdditionalSettingsCHR`.** Extra settings require their focused suite; a green default CHR run alone is not full 152-resource live acceptance. See [runner commands](orbstack-chr.md).

Only reviewed acquisition recipes for **7.24.5 and 7.25beta5 x86_64 CHR** exist. Base/container lanes have recorded clean-source evidence; the broader optional-package settings lane has focused development evidence only. ARM64, other versions/trains, physical hardware, packet processing, external-service behavior, arbitrary script/container execution, release publication and SDK-state migration are not certified by these runs.

## Collection behavior and limits

Collections generally create with PUT, read by `.id`, update with PATCH and delete by internal item ID. Import requires `*HEX`, not a name or numeric row index. SSH keys and IPsec policy groups are **replacement-only**; unsupported PATCH is never fabricated. Named replacement fields do not silently rename dependencies. Reviewed ownership/default/static guards reject unsafe built-in, dynamic or unrepresented objects where applicable.

Writable payloads come from configuration, never computed attributes/IDs. Optional + computed omission adopts server state, not a reset. Empty clearing and omitted read defaults are per-field approvals: for example, empty comments and supported CSV/list fields clear explicitly; pool `next_pool` omission reads as `none`. Invalid/malformed reads are atomic. Failed mutations retain recovery state; known allocated IDs survive failed create readback. CSV sets, RouterOS durations and selected IP/MAC/VLAN formats preserve equivalent configured spelling without hiding real drift.

Bridge PVID configuration requires explicit VLAN filtering; RouterOS hides it otherwise. Pools accept nonempty, non-overlapping IPv4 addresses/ranges, not IPv6/CIDR. DHCP networks use canonical IPv4 CIDR and typed DNS lists; static leases reject dynamic adoption. IPv4 routes require explicit destination/gateway and static metadata; no implicit default route or blackhole flag support is inferred. Option resources, matchers/sets, DHCP clients and relays exist separately, but the baseline server/network/lease schemas still exclude option attachments, lease scripts and time-format fields.

IPv4 firewall rules expose reviewed action subsets and same-chain static `place_before` ordering. Explicit empty means chain end; omitted placement is unmanaged. Unsafe/dynamic/self/cross-chain anchors fail closed. Action-specific target/mark requirements are maintained separately from remote inactive-field readback. No transaction/global ordering lock or traffic behavior certification is promised. See the individual [firewall references](../resources/ip_firewall_filter.md) for exact actions.

Sensitive preservation is per-field. SSH keys validate public-key format/fingerprint ownership and refuse administrator/authenticated-user keys; import cannot recover key material. Netwatch rows with unmanaged callbacks are rejected. VETH requires both verified matching extra publication and an enabled matching-version container package; no base CRUD paths are invented.

## Settings behavior and limits

All 46 settings resources read existing global objects and POST a reviewed fixed `/set` path. Identity derives from the menu path; import never accepts item IDs. **Destroy performs no reset, DELETE or device write.** Unknown/malformed/unauthorized settings reads preserve state and report an error instead of treating global settings as deleted. Lists/enums/bounds and secret omission/masking have reviewed semantics, not a generic fallback.

The additional settings cover cloud/DNS/SMB/SSH, PPP/server configuration, SNMP, clock/disk/LED/user settings, bandwidth/email, wireless/CAPsMAN, container and user-manager configuration. Package requirements and physical-board limitations are listed per resource. User-manager database relocation can reload/reset settings; dependent settings must explicitly depend on the database resource. Database contents/migration and physical LED acceptance remain uncertified.

## Historical records and remaining work

The [125-constructor historical table](resource-support.md), [DHCP option contract/evidence](dhcp-options-wave.md), and earlier sixteen-resource [hosted baseline evidence](../../schemas/hosted-chr-validation.json) retain their source-specific scope; their counts are not the current provider denominator. The eighteen original contracts/generated files were preserved against `1897fb32c03c43fb48f6ab3020cc3baf2e2b57b2`; later naming-only metadata changes do not expand their semantics.

Review the [remaining 80 constructor integrations](remaining-resources.md), additional fields/actions/hardware and corresponding tests independently. Offline maintenance generation/replay is implemented, but durable hosted receipt storage, snapshot-candidate live acceptance and release publication remain separate unfinished work. Do not infer those stages from source-test or offline-generated artifacts.

## Keeping reference pages current

```sh
python3 tools/docs/resources.py
python3 tools/docs/resources.py --check
python3 -m unittest discover -s tools/docs -v
```

The documentation renderer reads reviewed descriptors/catalogs and curated resource notes; it emits Markdown only, never provider schema/model/runtime code. Its check rejects stale, missing or extra resource pages.
