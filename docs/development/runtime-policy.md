# Maintained runtime policy

The current runtime registers **152 reviewed resources** (106 collections, including two replacement-only resources, and 46 settings) and zero data sources. Schemas/models are official-generator output; maintained methods enforce the resource-specific semantics. Exact fields and constraints are in the [resource references](../resources/index.md), with evidence boundaries in [coverage](coverage.md). Metadata retains runtime revision `framework-rest-preview-v10-reviewed-resources`; the earlier `framework-rest-preview-v3` IP-only and `v8` eighteen-resource descriptions are historical, not current limitations.

## REST and configuration

Each provider instance owns its transport, credentials, RouterOS version and sorted enabled-package inventory. Configure reads `/system/resource` and `/system/package` once; aliases never share global configuration/version state. Accounts need access to these metadata menus as well as managed resources. Every request uses its operation context; failed Configure closes idle connections and shares no client.

Only `http`/`https` are accepted. Native `api`/`apis`, embedded URL credentials, query/fragment URLs and malformed hosts are rejected. TLS verification defaults on, with minimum TLS 1.2; insecure is explicit opt-in. CA input is a validated PEM file and conflicts with insecure. Redirects are not followed. Configuration overrides environment fallbacks, including explicitly empty values; timeout defaults to 59 seconds and accepts 5–86400. See [configuration options](../index.md).

Paths/query parameters are structurally encoded. IDs are escaped as a single segment except literal `*`, as required by RouterOS. Collection IDs must be `*HEX`; path traversal/query injection and collection-target mutations with empty IDs are rejected. Credentials and request/response bodies are not logged. Response reads enforce a 4 MiB bound and I/O checks. Typed non-2xx errors omit response-body content. Empty/204 can satisfy no-output mutations; required object/array reads reject empty/null/malformed JSON.

## Version and package guards

Consumer-owned parsing supports stable patches, numeric beta/RC sequences, `_ab` nightlies and RouterOS channel suffixes. Beta precedes RC then stable; nightlies are comparable only within the same base/train. Only the RouterOS 7 REST family is implemented; unknown syntax/other majors fail Configure.

**The current warning classifier exempts only stable 7.24.5.** It still warns for 7.25beta5 even though historical configuration tests exist for that pinned target. This implementation detail is not a supported-version range or proof of compatibility for other architectures/packages. Recorded base/container and focused optional-package lanes are described separately in [coverage](coverage.md).

Enabled packages must include `routeros` and match the system version. Disabled packages confer no capabilities; missing/duplicate/invalid/mixed metadata fails Configure. VETH/container settings require container; legacy CAPsMAN/wireless CAP require wireless; user-manager settings require user-manager. Required packages are reviewed per resource, not inferred from inventory. Matching pinned package provisioning exists for owned disposable guests; package presence never authorizes new resources/fields.

Reviewed wire aliases/codecs handle RouterOS `.id`, hyphenated names, bool yes/no, decimal integers, typed lists, CSV sets and selected durations/automatic numeric values. No broad upstream drift dispatcher or speculative version-dependent renames are ported. New mappings require explicit policy and regression evidence; exact generated-schema/spec gates remain separate from live compatibility.

## Collection lifecycle and recovery

Most collections use PUT create, filtered GET by `.id`, PATCH update and DELETE item. Reviewed replacement fields use Framework replacement modifiers. SSH keys and IPsec policy groups are replacement-only and reject item PATCH. Import accepts validated internal IDs, not names or row numbers. Default/dynamic/static/action ownership rules are resource-specific and can reject adoption or mutation.

Payloads derive from configuration. Null optional values are omitted; computed attributes/IDs are never echoed. Explicit empty/false values follow reviewed clearing/codecs rather than a global missing-field fallback. Optional + computed omission retains/adopts server values. Selected equivalent IP/prefix/MAC/VLAN/CSV/duration readback preserves configuration spelling while real drift remains visible. IP `vrf` is read-only; bare-IP writes use explicit host prefixes.

An actual absent collection record or applicable typed 404 is distinct from authentication, transport, malformed or partial read failures. Refresh decodes and validates a temporary model before committing state. DELETE 404 is idempotent. Failed create readback retains a known allocated ID/configured state for recovery; failed update/delete retains state. Firewall movement uses reviewed fixed move commands with same-chain static anchor validation; no global ordering transaction/lock is promised.

## Settings lifecycle

All 46 settings resources GET an existing object and POST their fixed `/set` action for create/update. Import identity is path-derived (`/system/identity` → `system.identity`), not a `*HEX` item ID. Preflight reads precede writes, malformed reads are atomic, and failed mutations retain deterministic identity/recovery state.

**Destroy only relinquishes Terraform management: no DELETE, reset, restore or device write.** Settings read failures, including unavailable menus/empty objects, are diagnostics rather than deletion-repair signals. Explicit list/string clearing, enum/bound validation and equivalent CSV spelling are reviewed per field. User-manager database relocation can reload/reset settings; explicit dependency ordering is necessary when managing related resources together.

## Sensitive state and ownership

Provider credentials are sensitive and never replaced by unrelated remote password metadata. Multiple registered resources now have sensitive attributes; the old IP-only claim of no sensitive readback fields no longer applies. Approved fields preserve known secrets when RouterOS omits or returns recognized masks (`*****` / `**hidden**`), without accepting arbitrary malformed readback or inventing missing secret values. Fresh import cannot recover omitted secret material. Terraform sensitivity redacts display, not stored state; use a protected backend.

SSH keys require reviewed public-key formats, replacement-only mutation and SHA256 fingerprint ownership, including RouterOS padding. Administrator/authenticated-user keys and built-in user/groups are protected. Netwatch objects with unmanaged callbacks fail closed. No arbitrary script/container execution or administrator-key rotation is certified by configuration tests.

Schema versions remain zero. There is no historical SDK UpgradeState migration contract; import is not migration support, and `terraform state replace-provider` is not authorized by these tests.

## Evidence and maintenance boundary

Full offline mock/race coverage includes provider aliases, TLS/redirect/deadline/response bounds, strict IDs/types, package mismatches, primitive/CSV/duration validation, replacement, deletion repair, mutation recovery, atomic malformed refresh, sensitive omission and settings unmanagement. Recorded full-suite statement coverage is 80.8%.

Clean-source local/hosted configuration acceptance covers the preserved 125-constructor set on both pinned base/container lanes, with base VETH unavailable. The 27 additional settings have focused dirty-source evidence on both versions: 19 base / 26 matching optional-package resources, with physical LEDs unavailable on CHR. No consolidated current 152-resource, generated-maintenance-candidate, traffic, other-architecture or release certification is inferred. Static SDK guides remain pinned at `0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb` (MPL-2.0); upstream callbacks/scripts/tests are never executed.
