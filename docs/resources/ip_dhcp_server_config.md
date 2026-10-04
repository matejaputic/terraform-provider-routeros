# routeros_ip_dhcp_server_config

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/ip/dhcp-server/config`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Reads the existing settings object with GET; create and update POST the fixed `/ip/dhcp-server/config/set` action. Identity is `ip.dhcp-server.config`.

**Destroy only removes Terraform management. It does not delete, reset, or restore device settings.** A failed/malformed settings read is an error, not evidence that the singleton was deleted.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `accounting` | bool | Optional + computed | — |
| `id` | string | Computed | — |
| `radius_password` | string | Optional + computed | Choices: `"empty"`, `"same-as-user"` |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_ip_dhcp_server_config.example 'ip.dhcp-server.config'
```

Use the exact path-derived identity above, not an internal item ID. Import does not imply compatibility with historical SDK state.

## Verification scope

This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
