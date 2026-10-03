# routeros_ip_dhcp_server_lease

REST collection `/ip/dhcp-server/lease`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `address` | string | Required |
| `mac_address` | string | Required |
| `server` | string | Optional + computed |
| `comment` | string | Optional + computed |
| `disabled` | bool | Optional + computed |
| `block_access` | bool | Optional + computed |
| `dynamic` | bool | Computed |
| `status` | string | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_ip_dhcp_server_lease.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

See the [isolated DHCP/routing example](../../examples/dhcp-routing/main.tf). Time formats, DHCP options, scripts and actual client-exchange acceptance are not yet included.

Only static leases are managed. MAC addresses require 48 bits and equivalent configured spelling is preserved; imports adopt canonical remote spelling.
