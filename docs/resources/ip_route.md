# routeros_ip_route

REST collection `/ip/route`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `dst_address` | string | Required |
| `gateway` | string | Required |
| `distance` | int64 | Optional + computed |
| `routing_table` | string | Optional + computed |
| `scope` | int64 | Optional + computed |
| `target_scope` | int64 | Optional + computed |
| `comment` | string | Optional + computed |
| `disabled` | bool | Optional + computed |
| `dynamic` | bool | Computed |
| `active` | bool | Computed |
| `static` | bool | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_ip_route.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

Requires explicit canonical IPv4 destination CIDR and a gateway; no implicit default route. Only explicitly static records are managed. Distance 1–255, scopes 0–255. Blackhole/presence-sensitive flags and dynamic routes are not exposed. See [isolated example](../../examples/dhcp-routing/main.tf).
