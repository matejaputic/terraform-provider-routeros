# routeros_interface_vlan

REST collection `/interface/vlan`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `name` | string | Required; change replaces resource |
| `interface` | string | Required |
| `vlan_id` | int64 | Required |
| `comment` | string | Optional + computed |
| `disabled` | bool | Optional + computed |
| `use_service_tag` | bool | Optional + computed |
| `mtu` | int64 | Optional + computed |
| `running` | bool | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_interface_vlan.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).
