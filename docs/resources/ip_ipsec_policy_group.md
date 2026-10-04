# routeros_ip_ipsec_policy_group

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/ip/ipsec/policy/group`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Creates with PUT and reads by internal ID; changes require replacement rather than item PATCH. Destroy deletes the owned collection item.

Reviewed ownership checks can reject built-in, dynamic or otherwise unmanageable objects. Import is not blanket permission to modify every row. Failed refresh preserves prior state; a known allocated ID is retained for create recovery.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `id` | string | Computed | — |
| `name` | string | Required | Change requires replacement |

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_ip_ipsec_policy_group.example '*A'
```

Use an internal `*HEX` ID, not a name or numeric row index. Import does not imply compatibility with historical SDK state.

## Resource-specific notes

- Only a meaningful replacement-owned name is exposed; the purported comment field is unsupported by RouterOS.

## Verification scope

This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
