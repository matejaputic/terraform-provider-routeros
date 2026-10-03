# Snapshot adaptation and resource catalog (Step 3)

Run from the provider root. Python standard library adapter + official pinned HashiCorp generators; no custom Framework emitter, upstream execution or RouterOS mutation.

## Commands

```sh
# Offline baseline: pinned extracted sixteen-resource networking/DHCP/routing/firewall fixture.
bash tools/schema/generate.sh
make testschema

# Artifact exploration: preserve the exact input bytes and report their hash.
python3 tools/schema/normalize/normalize.py \
  --collections --input '<snapshot OpenAPI.json>' --output .local/adapt-example
python3 tools/schema/normalize/verify.py --output .local/adapt-example

# Provenance-qualified generation: first discover with current processing inputs.
python3 tools/schema/discover/discover.py run --extra
# Select a candidate's key/path/SHA from .local/discovery/manifest.json.
python3 tools/schema/normalize/normalize.py \
  --collections --manifest .local/discovery/manifest.json --key '<candidate key>' \
  --input '.local/discovery/<SHA>/<candidate schema path>' \
  --output '.local/adapt/<candidate key>'

# To update the maintained provider slice using that same verified candidate:
bash tools/schema/generate.sh \
  --manifest .local/discovery/manifest.json --key '<candidate key>' \
  --input '.local/discovery/<SHA>/<candidate schema path>'
```

Snapshot mode checks candidate identity/key, repository/version/channel/flavor/architecture/path, raw input SHA256, every saved metadata hash and **current** policy/runtime/tool/source fingerprints. Rediscover after producer changes; do not reuse a stale candidate key. Metadata cannot be overridden in manifest mode. The normalizer does not resolve HEAD or fetch mutable URLs.

Explicit `--input`/`--policy`/`--inspect` modes are developer artifact exploration, **not** a bypass for production discovery/freshness or release approval. They carry no discovered-candidate key unless a manifest was supplied. Nightly historical fixtures/replay demonstrate generation only, not freshness or device compatibility.

## Artifact bundle

| File | Contract |
| --- | --- |
| `upstream-input.json` | Exact unmodified bytes supplied to this run; full upstream artifact in snapshot mode. |
| `ip-address.openapi.json` | Separate schema-extraction OpenAPI, containing only approved resources; filename retained for bundle compatibility. |
| `generator-config.yml` | Explicit PUT create, GET-by-ID schema read, PATCH update, DELETE mappings. |
| `wire-descriptors.json` | Public/wire aliases, modes, types, codecs, validators/default ownership, lifecycle bindings, schema version and compatibility boundaries. |
| `adaptation-report.json` | Final bundle marker: input/normalized/config/descriptor/metadata hashes, producer pins/source hash, origin, adaptations, structural inventory/coverage and unreviewed suggestions. |

The report is written last. `verify.py` binds all four artifacts to its hashes, detecting partial or tampered bundles before generation. Input validation failure changes none of these artifacts. An I/O failure can leave partial files, but they cannot pass the bundle gate. File writes are atomic and serialized by a POSIX lock.

`generate.sh` stages official generator output, independently validates the Provider Code Specification, requires the exact sixteen Framework model/schema files, and only then replaces the specification/model. CLI exit success or warnings are **not** acceptance. Generated Go remains in `internal/generated/`; maintained Configure/CRUD/import code is not emitted or overwritten. Discovery stage receipts are not automatically advanced by this script; orchestration/evidence persistence belongs to Step 6.

## Conservative adaptation policy

- Resolve internal JSON-pointer refs, with bounds, cycle detection and no external fetches. Flatten supported object `allOf` recursively; union/array compositions and incompatible overlaps fail on approved input. Remove QueryOptions and command controls **before** resolving their unsupported unions.
- Inventory complete collection PUT + item GET/PATCH/DELETE sets, create/update writable differences, pending enum fields and unsupported compositions. POST commands are separately excluded. Singleton/settings, hardware and incomplete/read-only endpoints remain deferred. Structural completeness is not lifecycle approval.
- Sixteen resources are approved: IP address, bridge/bridge port, VLAN, interface list/member, IP pool, DHCP network/server/static lease, static route, permanent firewall address-list entries and reviewed ordered filter/NAT/mangle/raw rules. `generate.sh` always selects `--collections`; without that flag the standalone adapter retains the original IP-only exploration mode (not the complete provider artifact set). Unsupported unapproved endpoints are report findings, not partial public resources. Adding resources requires a reviewed policy, maintained lifecycle, registration and independent output/runtime gates.
- Remove `.query`, `.proplist`, `numbers`, `copy-from` and get-control state. Replace `.id` with one computed `id`; omit the item read's schema path parameter so the official extractor cannot invent a second ID. This is deliberately **schema-extraction** OpenAPI, not a REST routing contract. Wire metadata retains the ID path binding; runtime continues filtered collection GET for out-of-band deletion.
- `schemas/ip-address-policy.json` is the reviewed reference/live exception catalog, now `ip-address-curated-v3`. Repair incomplete/empty read responses from those definitions. Retain catalog fields absent from crawls; inspect suggestions are version-bound but never authoritative types/defaults or automatic additions.
- Required address/interface; optional+computed comment/disabled/network; computed ID, actual_interface, dynamic/invalid/slave/vrf. Preserve public booleans and reviewed yes/no codecs, validators and server-owned defaults. Type/boolean-domain drift, create-only approved fields without replacement policy and identifier collisions fail rather than silently changing contracts.
- Remove device-instance interface enums. Static domain enums, new lists/sets/numbers, sensitivity changes, ForceNew and state upgrades require an explicit maintained implementation and tests. The networking catalog explicitly implements int64, pool/DNS string lists/CSV, name replacement, reviewed scalar/list/boolean omission defaults and conditional bridge PVID reads; other changes still fail without maintained support. Unsupported candidate properties are only reported as unreviewed suggestions. No claim that all SDK schema features have been ported.
- Enforce stable snake_case, explicit `.id`/`actual-interface` aliases and Terraform/Go-symbol uniqueness. Schema version remains 0; SDK-state migration compatibility is **not claimed**.
- Provider configuration remains maintained Framework code with its Step 1 credentials/TLS/env handling. Password sensitivity is regression-tested; the adapter does not read ROS_/MIKROTIK_ credential environment variables. Raw upstream input is intentionally not redacted or modified; it is public upstream data, not provider connection state.

## Fixtures and evidence

Six compact fixtures preserve the actual info/components and IP-address operation objects from committed restraml blobs at `adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a`. They are **extracted/reserialized fixtures**, not whole upstream byte copies; provenance records both full-source and fixture SHA256. Default generation is offline and uses the additional sixteen-resource stable/base fixture built by `build_collections.py`; its separate provenance binds the original source and extracted bytes. Networking policy lives in `internal/catalog/collections.json`, is embedded in runtime and included in producer fingerprints. Full snapshot mode stores the complete original bytes separately.

```sh
python3 tools/schema/normalize/fixtures/build.py --repository-dir ../restraml
```

Real full-snapshot adaptation plus both official generators were verified for:

| RouterOS | Base structural CRUD candidates | Extra structural CRUD candidates |
| --- | ---: | ---: |
| 7.24.5 | 229 | 284 |
| 7.25beta5 | 231 | 286 |
| 7.25_ab434 (explicit historical replay) | 231 | 263 |

Every lane exposed exactly the same one reviewed resource and produced byte-identical Framework models/schema; no data sources or automatically supported additions. These are **generation results, not live acceptance results**. Step 1's 7.24.5 x86_64/base CHR evidence remains the only live baseline; no CHR lane was rerun or newly certified in Step 3.

Verification: 17 adapter/golden/failure tests + 6 independent specification-gate tests, alongside the 18 discovery tests. Go tests compare descriptors and policy to the actual Framework schema/validators/runtime aliases, assert provider password sensitivity, load protocol schema and run Terraform mock lifecycle. Go race tests, vet/build and repeat generation are additional gates.
