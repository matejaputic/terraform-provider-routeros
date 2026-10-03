# routeros_ip_dhcp_server_network

REST collection `/ip/dhcp-server/network`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `address` | string | Required |
| `gateway` | string | Optional + computed |
| `dns_server` | list(string) | Optional + computed |
| `dns_none` | bool | Optional + computed |
| `netmask` | int64 | Optional + computed |
| `domain` | string | Optional + computed |
| `comment` | string | Optional + computed |
| `dynamic` | bool | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_ip_dhcp_server_network.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

See the [isolated DHCP/routing example](../../examples/dhcp-routing/main.tf). Time formats, DHCP options, scripts and actual client-exchange acceptance are not yet included.

`address` requires canonical IPv4 CIDR; DNS is a typed IPv4 list and `[]` clears it. `dns_none = true` cannot be combined with nonempty DNS servers. Gateway is IPv4 and netmask 0–32.
