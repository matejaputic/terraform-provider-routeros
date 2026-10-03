# routeros_ip_dhcp_server

REST collection `/ip/dhcp-server`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `name` | string | Required; change replaces resource |
| `interface` | string | Required |
| `address_pool` | string | Required |
| `comment` | string | Optional + computed |
| `disabled` | bool | Optional + computed |
| `add_arp` | bool | Optional + computed |
| `conflict_detection` | bool | Optional + computed |
| `invalid` | bool | Computed |
| `dynamic` | bool | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_ip_dhcp_server.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

See the [isolated DHCP/routing example](../../examples/dhcp-routing/main.tf). Time formats, DHCP options, scripts and actual client-exchange acceptance are not yet included.
