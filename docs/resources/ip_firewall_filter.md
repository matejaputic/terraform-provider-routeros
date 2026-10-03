# routeros_ip_firewall_filter

REST collection `/ip/firewall/filter`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

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

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_ip_firewall_filter.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

Reviewed actions: accept/drop/log/passthrough/return. Other action-specific fields are deferred. `place_before` takes a same-chain static internal *HEX ID; explicit empty string moves to chain end. Omitted placement is unmanaged. Readback is the next same-chain ID; external moves are detected and repaired. Dynamic chains, self/missing/cross-chain anchors and malformed snapshots fail closed. No global transaction/ordering lock is promised. See the [isolated example](../../examples/firewall/main.tf).
