# routeros_ip_firewall_addr_list

REST collection `/ip/firewall/address-list`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `list` | string | Required |
| `address` | string | Required |
| `comment` | string | Optional + computed |
| `disabled` | bool | Optional + computed |
| `dynamic` | bool | Computed |
| `creation_time` | string | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_ip_firewall_addr_list.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

Permanent IPv4 address/canonical CIDR/ascending-range entries only. No DNS/IPv6/finite timeout support. Equivalent numeric configured spelling is retained; import adopts remote canonical spelling.
