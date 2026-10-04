# routeros_wifi_capsman

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/interface/wifi/capsman`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Reads the existing settings object with GET; create and update POST the fixed `/interface/wifi/capsman/set` action. Identity is `interface.wifi.capsman`.

**Destroy only removes Terraform management. It does not delete, reset, or restore device settings.** A failed/malformed settings read is an error, not evidence that the singleton was deleted.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `ca_certificate` | string | Optional + computed | — |
| `certificate` | string | Optional + computed | — |
| `enabled` | bool | Optional + computed | — |
| `id` | string | Computed | — |
| `interfaces` | string | Optional + computed | Comma-separated set in a Terraform string, not an HCL list |
| `require_peer_certificate` | bool | Optional + computed | — |
| `upgrade_policy` | string | Optional + computed | Choices: `"none"`, `"suggest-same-version"`, `"require-same-version"` |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_wifi_capsman.example 'interface.wifi.capsman'
```

Use the exact path-derived identity above, not an internal item ID. Import does not imply compatibility with historical SDK state.

## Verification scope

Focused dirty-source configuration lifecycles passed on both pinned x86_64/base targets and matching optional-package targets. Consolidated clean-source/hosted and maintenance-candidate certification of these additional settings is not recorded.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
