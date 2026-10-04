# routeros_ip_tftp_settings

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/ip/tftp/settings`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Reads the existing settings object with GET; create and update POST the fixed `/ip/tftp/settings/set` action. Identity is `ip.tftp.settings`.

**Destroy only removes Terraform management. It does not delete, reset, or restore device settings.** A failed/malformed settings read is an error, not evidence that the singleton was deleted.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `id` | string | Computed | — |
| `max_block_size` | int64 | Optional + computed | Bounds: 512 to 65535 |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_ip_tftp_settings.example 'ip.tftp.settings'
```

Use the exact path-derived identity above, not an internal item ID. Import does not imply compatibility with historical SDK state.

## Verification scope

This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
