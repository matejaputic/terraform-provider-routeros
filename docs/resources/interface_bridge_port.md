# routeros_interface_bridge_port

REST collection `/interface/bridge/port`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `bridge` | string | Required |
| `interface` | string | Required |
| `comment` | string | Optional + computed |
| `disabled` | bool | Optional + computed |
| `pvid` | int64 | Optional + computed |
| `edge` | string | Optional + computed |
| `hw` | bool | Optional + computed |
| `dynamic` | bool | Computed |
| `inactive` | bool | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_interface_bridge_port.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).
