# Terraform Provider RouterOS (Framework preview)

Development identity: `matejaputic/routeros`. Registry authorization and publishing are not yet configured. This repository retains its existing history and origin.

Sixteen resources are implemented and live-tested: `routeros_ip_address`, `routeros_interface_bridge`, `routeros_interface_bridge_port`, `routeros_interface_vlan`, `routeros_interface_list`, `routeros_interface_list_member`, `routeros_ip_pool`, `routeros_ip_dhcp_server_network`, `routeros_ip_dhcp_server`, `routeros_ip_dhcp_server_lease`, `routeros_ip_route`, `routeros_ip_firewall_addr_list`, `routeros_ip_firewall_filter`, `routeros_ip_firewall_nat`, `routeros_ip_firewall_mangle`, and `routeros_ip_firewall_raw`. See [coverage/limitations](docs/development/coverage.md) and a [dependent networking example](examples/networking/main.tf). Each exposes a reviewed field subset. No data sources are exposed; this is not yet full reference-provider parity or SDK-state migration compatibility. REST (`http`/`https`) only; native `api`/`apis` transports are explicitly unsupported.

## Development

Go 1.25.8 and Framework v1.19.0 are declared in `go.mod`. The official OpenAPI generator is pinned to v0.3.0 and Framework generator to v0.4.1. Generation is tech preview.

```sh
bash tools/schema/generate.sh
TF_ACC_TERRAFORM_VERSION=1.14.0 go test -race ./...
go vet ./...
go build .
```

The mock lifecycle test uses a real Terraform 1.14.0 CLI and tests create, update, import, delete and an empty second plan. It does **not** prove RouterOS compatibility. Generated schemas/models live in `internal/generated/`; maintained provider and lifecycle methods never get overwritten. Step 3 connects [immutable discovery](tools/schema/discover/README.md) to a [snapshot-fed adapter and resource catalog](tools/schema/normalize/README.md). Default generation uses a pinned extracted upstream fixture; manifest mode verifies full snapshot/hash/producer identity before official generation. Only the reviewed IP-address resource is exposed; other endpoints remain coverage findings.

`schemas/provenance.json` records tool/policy versions and the MPL-2.0 reference source SHA. Reference behavior is adapted from terraform-routeros/terraform-provider-routeros; no SDK callbacks are used. Attributes retain boolean flags, required address/interface, computed actual_interface/dynamic/invalid/slave, optional+computed network and RouterOS `.id` imports. Framework optional+computed comment/disabled deliberately retain server defaults when omitted. `vrf` is computed-only: live RouterOS 7.24.5 rejects it as a writable IP-address parameter, unlike the reference SDK schema. This reviewed exception is recorded in `schemas/ip-address-policy.json`; exact SDK state compatibility is not claimed.

## Snapshot discovery

```sh
python3 tools/schema/discover/discover.py run
python3 tools/schema/discover/discover.py run --extra --backfill 1
make testdiscovery
```

Discovery resolves upstream once, reads full-SHA artifacts, validates versions/provenance, hashes semantic inputs and records separate successful-stage checkpoints. Optional nightly discovery rejects stale/mixed data; explicit `--replay` permits historical freshness/regression overrides only. Outputs/checkpoints stay under ignored `.local/`. Snapshot-fed normalization/generation is implemented, but scheduled orchestration, acceptance expansion and publication remain later steps.

## Snapshot adaptation

```sh
make testschema
python3 tools/schema/normalize/normalize.py --input '<snapshot OpenAPI.json>' --output .local/adapt-example
```

The adapter preserves input bytes, resolves supported refs/object compositions, strips query controls, repairs curated responses and writes normalized OpenAPI, generator config, wire descriptors and adaptation/coverage reports. Unsupported approved compositions, collisions, type/domain drift and unreviewed lifecycle changes fail. Both official generators and independent output gates were verified against stable/base+extra, beta/base+extra and historical nightly/base+extra snapshots; this is not live compatibility evidence for additional lanes. See [full commands and boundaries](tools/schema/normalize/README.md).

## Configuration

Use `hosturl`, `username`, sensitive `password`, `ca_certificate` (PEM file path), `insecure` (explicit opt-in), and `rest_timeout` (seconds, default 59, minimum 5). Configuration overrides environment variables, including explicitly empty strings.

| Option | Environment fallback, in priority order |
| --- | --- |
| hosturl | ROS_HOSTURL, MIKROTIK_HOST |
| username | ROS_USERNAME, MIKROTIK_USER |
| password | ROS_PASSWORD, MIKROTIK_PASSWORD |
| ca_certificate | ROS_CA_CERTIFICATE, MIKROTIK_CA_CERTIFICATE |
| insecure | ROS_INSECURE, MIKROTIK_INSECURE |

TLS verification defaults on; CA and insecure cannot be combined. Credentials are never logged; redirects are not followed. Each REST operation uses its own context. HTTP is supported for isolated test networks but does not encrypt credentials.

## Coverage-first implementation

The foundational networking batch is complete: six additional resources use a maintained reusable collection lifecycle with official generated schemas/models, typed int64/CSV-list codecs, reviewed defaults/conditional reads, name replacement and built-in/dynamic ownership checks. All six pass real Terraform mock CRUD/import and disposable CHR CRUD/import/update/empty-plan acceptance. Pool out-of-band recreation and name replacement also pass. The DHCP/static-routing batch adds four more live-tested lifecycles, typed DNS lists/clearing, canonical IPv4/MAC validation and static-only ownership guards. Static IPv4 firewall address lists and a reviewed filter subset now pass mock/live acceptance, including explicit same-chain ordering and external reorder repair. NAT/mangle/raw now add three more tested ordered lifecycles with reviewed targets, marking and notrack actions. Next priority is broader reviewed matchers/actions and additional subsystems; publication automation remains deferred.

## Step 4 runtime hardening

See [runtime policy and evidence](docs/development/runtime-policy.md). Configure keeps version and verified enabled-package metadata per client, rejects unsupported version families/mixed packages and warns for unreviewed versions. REST uses owned TLS transports, operation contexts, bounded/redacted responses and strict IDs. Failed refresh preserves state; failed post-create reads retain a known recovery ID. Concurrent aliases and the disposable 7.24.5 x86_64/base lifecycle pass. Extra packages do not authorize additional resources; SDK migration and generic masked resource-secret handling remain unclaimed.

## Step 1 status and release safety

Implemented generated schema/model integration, maintained REST lifecycle, provider configuration, typed/redacted errors, wire aliases, mock lifecycle tests and independent spec shape checks. Protocol 6 matches the Registry manifest.

**Step 1 vertical slice verified:** live disposable CHR 7.24.5 x86_64 on OrbStack/QEMU passes create/read/update/import/delete, reference-style typed state checks, empty second plan/apply, and out-of-band deletion/recreation with a new unique ID. Go 1.25.8 race tests/vet/build and deterministic generation pass. Provider Configure reads RouterOS version and actual enabled packages into each client independently. General version-drift adaptation and historical-state migration remain future work; no migration compatibility is claimed. Do not use this preview on production routers or run `terraform state replace-provider` based on these tests.

For the reproducible local fixture (not a native OrbStack ISO machine), see [OrbStack CHR setup](docs/development/orbstack-chr.md):

```sh
python3 tools/chr/chr.py start
python3 tools/chr/chr.py test
python3 tools/chr/chr.py stop
```

The fixture pins the official x86 CHR 7.24.5 archive/disk hashes and uses QEMU/TCG inside an unprivileged OrbStack Docker container. Ports bind only to loopback, guest egress is restricted, credentials are random and stored only under ignored `.local/chr/`. `stop` removes credentials and the mutable disk.

Release workflow is intentionally disabled pending acceptance and promotion gates. No stable/nightly release is authorized by this scaffold implementation. CI only validates source and mocks; no signing or router mutation secrets are used.
