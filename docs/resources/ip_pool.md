# routeros_ip_pool

REST collection `/ip/pool`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `name` | string | Required; change replaces resource |
| `ranges` | list(string) | Required |
| `comment` | string | Optional + computed |
| `next_pool` | string | Optional + computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_ip_pool.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

`ranges` is a nonempty, non-overlapping list of IPv4 addresses or ascending `start-end` ranges (no IPv6/CIDR). Missing `next_pool` reads as `none`; use canonical order when importing existing ranges.
