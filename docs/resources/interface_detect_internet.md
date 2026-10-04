# routeros_interface_detect_internet

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/interface/detect-internet`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Reads the existing settings object with GET; create and update POST the fixed `/interface/detect-internet/set` action. Identity is `interface.detect-internet`.

**Destroy only removes Terraform management. It does not delete, reset, or restore device settings.** A failed/malformed settings read is an error, not evidence that the singleton was deleted.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `detect_interface_list` | string | Optional + computed | — |
| `id` | string | Computed | — |
| `internet_interface_list` | string | Optional + computed | — |
| `lan_interface_list` | string | Optional + computed | — |
| `wan_interface_list` | string | Optional + computed | — |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_interface_detect_internet.example 'interface.detect-internet'
```

Use the exact path-derived identity above, not an internal item ID. Import does not imply compatibility with historical SDK state.

## Verification scope

Focused dirty-source configuration lifecycles passed on both pinned x86_64/base targets and matching optional-package targets. Consolidated clean-source/hosted and maintenance-candidate certification of these additional settings is not recorded.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
