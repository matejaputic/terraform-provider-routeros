# routeros_ip_dhcp_server

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/ip/dhcp-server`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Creates with PUT, reads the collection filtered by internal ID, updates with PATCH, and deletes the item with DELETE. Replacement-marked fields recreate the item rather than rename it in place.

Reviewed ownership checks can reject built-in, dynamic or otherwise unmanageable objects. Import is not blanket permission to modify every row. Failed refresh preserves prior state; a known allocated ID is retained for create recovery.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `add_arp` | bool | Optional + computed | Reviewed omitted read value: `"false"` |
| `address_pool` | string | Required | — |
| `comment` | string | Optional + computed | Reviewed omitted read value: `""` |
| `conflict_detection` | bool | Optional + computed | Reviewed omitted read value: `"true"` |
| `disabled` | bool | Optional + computed | — |
| `dynamic` | bool | Computed | — |
| `id` | string | Computed | — |
| `interface` | string | Required | — |
| `invalid` | bool | Computed | — |
| `name` | string | Required | Change requires replacement |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_ip_dhcp_server.example '*A'
```

Use an internal `*HEX` ID, not a name or numeric row index. Import does not imply compatibility with historical SDK state.

## Resource-specific notes

- The reviewed server fields do not include lease scripts, time-format settings or option attachments. Client/server option definitions, option sets/matchers, relays and DHCP-client resources are available separately; their availability does not add attributes to this server schema.
- `address_pool` is required. Omitted `add_arp` reads as false and omitted `conflict_detection` as true. Live acceptance keeps the server disabled and does not certify actual DHCP exchanges.

## Verification scope

This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
