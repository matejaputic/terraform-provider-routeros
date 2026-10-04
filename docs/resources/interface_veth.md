# routeros_interface_veth

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/interface/veth`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Creates with PUT, reads the collection filtered by internal ID, updates with PATCH, and deletes the item with DELETE. Replacement-marked fields recreate the item rather than rename it in place.

Reviewed ownership checks can reject built-in, dynamic or otherwise unmanageable objects. Import is not blanket permission to modify every row. Failed refresh preserves prior state; a known allocated ID is retained for create recovery.

**Requires the enabled `container` package matching the RouterOS system version.** Package presence does not authorize unlisted fields or certify other targets.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `address` | string | Required | Comma-separated set in a Terraform string, not an HCL list |
| `comment` | string | Optional + computed | Reviewed omitted read value: `""` |
| `dhcp` | bool | Optional + computed | — |
| `disabled` | bool | Optional + computed | Reviewed omitted read value: `"false"` |
| `gateway` | string | Optional + computed | Validation: `ip4` |
| `id` | string | Computed | — |
| `name` | string | Required | Change requires replacement |
| `running` | bool | Computed | — |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_interface_veth.example '*A'
```

Use an internal `*HEX` ID, not a name or numeric row index. Import does not imply compatibility with historical SDK state.

## Resource-specific notes

- Requires a hash-bound matching extra publication during generation and an enabled matching-version container package at runtime. Base publication paths are never fabricated. Configuration testing does not execute containers.

## Verification scope

This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
