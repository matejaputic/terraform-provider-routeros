# routeros_ip_dhcp_server_network

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/ip/dhcp-server/network`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Creates with PUT, reads the collection filtered by internal ID, updates with PATCH, and deletes the item with DELETE. Replacement-marked fields recreate the item rather than rename it in place.

Reviewed ownership checks can reject built-in, dynamic or otherwise unmanageable objects. Import is not blanket permission to modify every row. Failed refresh preserves prior state; a known allocated ID is retained for create recovery.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `address` | string | Required | — |
| `comment` | string | Optional + computed | Reviewed omitted read value: `""` |
| `dns_none` | bool | Optional + computed | Reviewed omitted read value: `"false"` |
| `dns_server` | list(string) | Optional + computed | Reviewed omitted read value: `""` |
| `domain` | string | Optional + computed | Reviewed omitted read value: `""` |
| `dynamic` | bool | Computed | — |
| `gateway` | string | Optional + computed | — |
| `id` | string | Computed | — |
| `netmask` | int64 | Optional + computed | — |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_ip_dhcp_server_network.example '*A'
```

Use an internal `*HEX` ID, not a name or numeric row index. Import does not imply compatibility with historical SDK state.

## Resource-specific notes

- `address` requires canonical IPv4 CIDR, `gateway` and DNS entries are IPv4, and `netmask` is 0–32. `dns_server = []` clears the list; `dns_none = true` cannot be combined with nonempty DNS servers. Omitted DNS/domain read as empty and omitted `dns_none` as false.
- DHCP option resources are available separately, but option attachments and script/time-format settings are not attributes of this network resource.

## Verification scope

This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
