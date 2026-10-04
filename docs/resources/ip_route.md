# routeros_ip_route

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/ip/route`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Creates with PUT, reads the collection filtered by internal ID, updates with PATCH, and deletes the item with DELETE. Replacement-marked fields recreate the item rather than rename it in place.

Reviewed ownership checks can reject built-in, dynamic or otherwise unmanageable objects. Import is not blanket permission to modify every row. Failed refresh preserves prior state; a known allocated ID is retained for create recovery.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `active` | bool | Computed | Reviewed omitted read value: `"false"` |
| `comment` | string | Optional + computed | Reviewed omitted read value: `""` |
| `disabled` | bool | Optional + computed | — |
| `distance` | int64 | Optional + computed | — |
| `dst_address` | string | Required | — |
| `dynamic` | bool | Computed | Reviewed omitted read value: `"false"` |
| `gateway` | string | Required | — |
| `id` | string | Computed | — |
| `routing_table` | string | Optional + computed | — |
| `scope` | int64 | Optional + computed | — |
| `static` | bool | Computed | — |
| `target_scope` | int64 | Optional + computed | — |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_ip_route.example '*A'
```

Use an internal `*HEX` ID, not a name or numeric row index. Import does not imply compatibility with historical SDK state.

## Resource-specific notes

- Requires explicit canonical IPv4 destination CIDR and gateway; no implicit default route. Only explicitly static records are managed; missing static metadata fails closed. Distance is 1–255 and scopes are 0–255. Blackhole/presence-sensitive flags and dynamic routes are not exposed.

## Verification scope

This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
