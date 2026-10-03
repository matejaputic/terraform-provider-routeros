# routeros_interface_bridge

REST collection `/interface/bridge`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `name` | string | Required; change replaces resource |
| `comment` | string | Optional + computed |
| `disabled` | bool | Optional + computed |
| `protocol_mode` | string | Optional + computed |
| `vlan_filtering` | bool | Optional + computed |
| `pvid` | int64 | Optional + computed |
| `running` | bool | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_interface_bridge.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).

Configuring `pvid` requires explicit `vlan_filtering = true`; RouterOS hides PVID otherwise. VLAN filtering can disrupt connectivity: do not experiment on a management bridge.
