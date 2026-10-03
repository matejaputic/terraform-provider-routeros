// SPDX-License-Identifier: MPL-2.0
package catalog

// Curated from terraform-routeros/terraform-provider-routeros at
// 0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb (MPL-2.0), resource_ip_address.go.
// Metadata is deliberately separate from the public Terraform schema.
const IPAddressPath = "/ip/address"
const IDKey = ".id"

var IPAddressWireNames = map[string]string{
	"id": ".id", "actual_interface": "actual-interface", "address": "address",
	"interface": "interface", "comment": "comment", "disabled": "disabled",
	"dynamic": "dynamic", "invalid": "invalid", "network": "network", "slave": "slave", "vrf": "vrf",
}
