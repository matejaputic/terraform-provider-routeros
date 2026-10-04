# routeros_user_manager_database

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/user-manager/database`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Reads the existing settings object with GET; create and update POST the fixed `/user-manager/database/set` action. Identity is `user-manager.database`.

**Destroy only removes Terraform management. It does not delete, reset, or restore device settings.** A failed/malformed settings read is an error, not evidence that the singleton was deleted.

**Requires the enabled `user-manager` package matching the RouterOS system version.** Package presence does not authorize unlisted fields or certify other targets.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `db_path` | string | Optional + computed | — |
| `id` | string | Computed | — |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_user_manager_database.example 'user-manager.database'
```

Use the exact path-derived identity above, not an internal item ID. Import does not imply compatibility with historical SDK state.

## Resource-specific notes

- Changing `db_path` reloads the database and can reset user-manager settings and advanced options. Manage this resource first and give dependent user-manager resources an explicit `depends_on`. Database contents, relocation safety and migration compatibility are not certified.

## Verification scope

Focused dirty-source configuration lifecycles passed on both pinned x86_64 targets with matching container, wireless and user-manager packages. This resource is unavailable in base lanes without its required package. Consolidated clean-source/hosted and maintenance-candidate certification of these additional settings is not recorded.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
