# routeros_interface_list

REST collection `/interface/list`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.

| Attribute | Type | Mode |
| --- | --- | --- |
| `id` | string | Computed |
| `name` | string | Required; change replaces resource |
| `comment` | string | Optional + computed |
| `include` | string | Optional + computed |
| `exclude` | string | Optional + computed |
| `builtin` | bool | Computed |
| `dynamic` | bool | Computed |

Unconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.

```sh
terraform import routeros_interface_list.example '*A'
```

No SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).
