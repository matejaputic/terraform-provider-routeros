# routeros_ip_dns

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/ip/dns`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Reads the existing settings object with GET; create and update POST the fixed `/ip/dns/set` action. Identity is `ip.dns`.

**Destroy only removes Terraform management. It does not delete, reset, or restore device settings.** A failed/malformed settings read is an error, not evidence that the singleton was deleted.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `allow_remote_requests` | bool | Optional + computed | — |
| `cache_size` | int64 | Optional + computed | Bounds: 64 to 4294967295 |
| `id` | string | Computed | — |
| `max_concurrent_queries` | int64 | Optional + computed | Bounds: 1 to 65535 |
| `max_concurrent_tcp_sessions` | int64 | Optional + computed | Bounds: 1 to 65535 |
| `max_udp_packet_size` | int64 | Optional + computed | Bounds: 512 to 65535 |
| `servers` | string | Optional + computed | Comma-separated set in a Terraform string, not an HCL list |
| `verify_doh_cert` | bool | Optional + computed | — |
| `vrf` | string | Optional + computed | — |

Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.

Computed attributes and IDs are never mutation inputs.

## Import

```sh
terraform import routeros_ip_dns.example 'ip.dns'
```

Use the exact path-derived identity above, not an internal item ID. Import does not imply compatibility with historical SDK state.

## Verification scope

Focused dirty-source configuration lifecycles passed on both pinned x86_64/base targets and matching optional-package targets. Consolidated clean-source/hosted and maintenance-candidate certification of these additional settings is not recorded.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
