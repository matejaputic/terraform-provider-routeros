# Terraform Provider for RouterOS

This preview exposes **152 canonical resource constructors**: **106 collection resources** and **46 global/settings resources**, with **zero data sources**. Schemas and models use HashiCorp's official OpenAPI/Framework generators; reviewed maintained code supplies REST lifecycles, ownership checks, validation and failure recovery. The static reference inventory contains 232 constructors, of which 80 remain unimplemented; aliases are not additional resources.

- [All resource references](resources/index.md): exact reviewed attributes, import identities, package requirements and special lifecycles.
- [Coverage and evidence](development/coverage.md): current registration versus historical configuration-test scope.
- [Runtime policy](development/runtime-policy.md): transport, version/package checks, state and sensitive values.
- [Remaining resource support](development/remaining-resources.md): additional settings and outstanding constructor work.
- [Continuous maintenance](development/maintenance.md) and [hosted/release status](development/hosted-checkpoint.md).
- [Disposable CHR testing](development/orbstack-chr.md).

## Installation status

Development identity: `registry.terraform.io/matejaputic/routeros`. Registry publication and stable release/signing authorization are not configured; use a locally built development provider, not an assumed published Registry version. This preview does not claim SDK-state migration or full reference-provider field parity. Review configuration examples on a disposable/owned test router before using them on existing networks.

## Provider configuration

Only RouterOS REST over `http`/`https` is supported; native `api`/`apis` transports are rejected. Prefer verified HTTPS: HTTP does not encrypt credentials. Each provider alias has an independent client, credentials, version and enabled-package inventory. Accounts need read access to `/system/resource` and `/system/package` as well as the managed menus.

| Option | Meaning | Environment fallbacks, in priority order |
| --- | --- | --- |
| `hosturl` | Router URL; required unless supplied by environment | `ROS_HOSTURL`, `MIKROTIK_HOST` |
| `username` | Router account; required unless supplied by environment | `ROS_USERNAME`, `MIKROTIK_USER` |
| `password` | Sensitive credential | `ROS_PASSWORD`, `MIKROTIK_PASSWORD` |
| `ca_certificate` | PEM CA certificate file path | `ROS_CA_CERTIFICATE`, `MIKROTIK_CA_CERTIFICATE` |
| `insecure` | Explicit TLS-verification opt-out; defaults false | `ROS_INSECURE`, `MIKROTIK_INSECURE` |
| `rest_timeout` | Request timeout, 5–86400 seconds; default 59 | None |

Explicit configuration overrides environment values, including explicitly empty strings. CA and insecure cannot be combined. Credentials and response bodies are not logged; sensitivity redacts normal Terraform output but does not encrypt state.

## Resource behavior

Collections generally use PUT/GET/PATCH/DELETE and import an internal `*HEX` ID; SSH keys and IPsec policy groups are replacement-only. Settings use GET-object/POST-`/set` and import a path-derived identity such as `system.identity`; **destroy only unmanages settings and does not reset the router**. Optional + computed fields adopt server values when omitted; computed attributes are never writes. Per-resource notes describe clearing, defaults, constraints and ownership exclusions.

## Verification boundaries

Recorded clean-source/hosted configuration tests cover the preserved 125-constructor set on 7.24.5 and 7.25beta5 x86_64 base/container lanes, with VETH unavailable on base. The 27 additional settings have focused mock/race and dirty-source live evidence: 19 available on base, 26 with matching optional packages, and physical LEDs unavailable on CHR. Registration does not imply all 152 resources were live-tested together or that a release/generated maintenance candidate passed corresponding-target acceptance.
