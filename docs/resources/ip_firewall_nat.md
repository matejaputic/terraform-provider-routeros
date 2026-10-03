# routeros_ip_firewall_nat

REST collection `/ip/firewall/nat`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `chain` | string | Required |
| `action` | string | Required |
| `src_address_list` | string | Optional + computed |
| `dst_address_list` | string | Optional + computed |
| `comment` | string | Optional + computed |
| `disabled` | bool | Optional + computed |
| `log` | bool | Optional + computed |
| `dynamic` | bool | Computed |
| `invalid` | bool | Computed |
| `place_before` | string | Optional + computed |
| `to_addresses` | string | Optional + computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_ip_firewall_nat.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

Reviewed static ordered rule subset. `place_before` uses a same-chain internal *HEX anchor; explicit empty moves to chain end, omitted placement is unmanaged. See [action requirements and live evidence](../development/coverage.md) and [disabled isolated examples](../../examples/firewall-actions/main.tf). No packet-processing certification or full reference parity.
