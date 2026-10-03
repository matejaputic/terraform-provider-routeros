# routeros_ip_address (Resource)

Preview Framework resource for `/ip/address` over REST. Live acceptance passes on disposable RouterOS 7.24.5 x86_64 CHR.

Required: `address` (IP or CIDR), `interface` (nonempty name, not a device-derived enum).

Optional and computed: `comment`, `disabled` (boolean), `network` (IP). Omitted fields retain server defaults.

Computed: `id`, `actual_interface`, `dynamic`, `invalid`, `slave`, `vrf`. `dynamic`, `invalid`, and `slave` are booleans. `vrf` is computed-only because the supported RouterOS baseline rejects it as writable. The wire `.id` is represented by a single computed Terraform `id`.

```hcl
resource "routeros_ip_address" "example" {
  address   = "192.0.2.1/24"
  interface = "ether1"
  disabled  = false
}
```

Import with the RouterOS ID:

```sh
terraform import routeros_ip_address.example '*2'
```

Historical SDK state upgrades are not implemented; do not infer migration support from import support.
