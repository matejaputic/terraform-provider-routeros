# routeros_user_manager_advanced

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/user-manager/advanced`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Reads the existing settings object with GET; create and update POST the fixed `/user-manager/advanced/set` action. Identity is `user-manager.advanced`.

**Destroy only removes Terraform management. It does not delete, reset, or restore device settings.** A failed/malformed settings read is an error, not evidence that the singleton was deleted.

**Requires the enabled `user-manager` package matching the RouterOS system version.** Package presence does not authorize unlisted fields or certify other targets.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `id` | string | Computed | — |
| `paypal_allow` | bool | Optional + computed | — |
| `paypal_password` | string | Optional + computed | Sensitive; protect Terraform state; Preserves known secret on omitted/masked readback |
| `paypal_signature` | string | Optional + computed | Sensitive; protect Terraform state; Preserves known secret on omitted/masked readback |
| `paypal_use_sandbox` | bool | Optional + computed | — |
| `paypal_user` | string | Optional + computed | Sensitive; protect Terraform state; Preserves known secret on omitted/masked readback |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

Terraform sensitivity redacts normal display; it does **not** encrypt state. Use a protected state backend. Secret preservation is per-field, not a generic substitution for any masked value.

## Import

```sh
terraform import routeros_user_manager_advanced.example 'user-manager.advanced'
```

Use the exact path-derived identity above, not an internal item ID. Import does not imply compatibility with historical SDK state.

## Resource-specific notes

- When managing a database-path change, use `depends_on = [routeros_user_manager_database.example]` so database reload precedes advanced-option changes. External payment-provider behavior is not certified.

## Verification scope

Focused dirty-source configuration lifecycles passed on both pinned x86_64 targets with matching container, wireless and user-manager packages. This resource is unavailable in base lanes without its required package. Consolidated clean-source/hosted and maintenance-candidate certification of these additional settings is not recorded.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
