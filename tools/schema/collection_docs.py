#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Generate field tables for the reviewed networking subset, not runtime code."""

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
policies = json.loads((ROOT / "internal/catalog/collections.json").read_bytes())
for policy in policies:
    name = policy["resource_name"]
    text = f"# routeros_{name}\n\nREST collection `{policy['wire_path']}`. Maintained Framework CRUD and internal-ID import; reviewed field subset, not full reference-provider parity. Live-tested on 7.24.5 x86_64/base CHR.\n\n| Attribute | Type | Mode |\n| --- | --- | --- |\n"
    for field in policy["attributes"]:
        kind = {
            "array": "list(string)",
            "integer": "int64",
            "boolean": "bool",
            "string": "string",
        }[field["type"]]
        mode = {
            "required": "Required",
            "computed_optional": "Optional + computed",
            "computed": "Computed",
        }[field["mode"]]
        if field["force_new"]:
            mode += "; change replaces resource"
        text += f"| `{field['name']}` | {kind} | {mode} |\n"
    text += (
        "\nUnconfigured writable fields retain server defaults; explicit empty comments and false booleans are sent. Built-in/dynamic objects are not managed.\n\n```sh\nterraform import routeros_"
        + name
        + ".example '*A'\n```\n\nNo SDK-state migration is claimed. See [coverage and caveats](../development/coverage.md) and the [dependent networking example](../../examples/networking/main.tf).\n"
    )
    if name == "interface_bridge":
        text += "\nConfiguring `pvid` requires explicit `vlan_filtering = true`; RouterOS hides PVID otherwise. VLAN filtering can disrupt connectivity: do not experiment on a management bridge.\n"
    if name.startswith("ip_dhcp_server"):
        text += "\nSee the [isolated DHCP/routing example](../../examples/dhcp-routing/main.tf). Time formats, DHCP options, scripts and actual client-exchange acceptance are not yet included.\n"
    if name == "ip_dhcp_server_network":
        text += "\n`address` requires canonical IPv4 CIDR; DNS is a typed IPv4 list and `[]` clears it. `dns_none = true` cannot be combined with nonempty DNS servers. Gateway is IPv4 and netmask 0–32.\n"
    if name == "ip_dhcp_server_lease":
        text += "\nOnly static leases are managed. MAC addresses require 48 bits and equivalent configured spelling is preserved; imports adopt canonical remote spelling.\n"
    if name == "ip_route":
        text += "\nRequires explicit canonical IPv4 destination CIDR and a gateway; no implicit default route. Only explicitly static records are managed. Distance 1–255, scopes 0–255. Blackhole/presence-sensitive flags and dynamic routes are not exposed. See [isolated example](../../examples/dhcp-routing/main.tf).\n"
    if name == "ip_firewall_addr_list":
        text += "\nPermanent IPv4 address/canonical CIDR/ascending-range entries only. No DNS/IPv6/finite timeout support. Equivalent numeric configured spelling is retained; import adopts remote canonical spelling.\n"
    if name == "ip_firewall_filter":
        text += "\nReviewed actions: accept/drop/log/passthrough/return. Other action-specific fields are deferred. `place_before` takes a same-chain static internal *HEX ID; explicit empty string moves to chain end. Omitted placement is unmanaged. Readback is the next same-chain ID; external moves are detected and repaired. Dynamic chains, self/missing/cross-chain anchors and malformed snapshots fail closed. No global transaction/ordering lock is promised. See the [isolated example](../../examples/firewall/main.tf).\n"
    if name in ("ip_firewall_nat", "ip_firewall_mangle", "ip_firewall_raw"):
        text += "\nReviewed static ordered rule subset. `place_before` uses a same-chain internal *HEX anchor; explicit empty moves to chain end, omitted placement is unmanaged. See [action requirements and live evidence](../development/coverage.md) and [disabled isolated examples](../../examples/firewall-actions/main.tf). No packet-processing certification or full reference parity.\n"
    if name == "ip_pool":
        text += "\n`ranges` is a nonempty, non-overlapping list of IPv4 addresses or ascending `start-end` ranges (no IPv6/CIDR). Missing `next_pool` reads as `none`; use canonical order when importing existing ranges.\n"
    (ROOT / "docs/resources" / f"{name}.md").write_text(text)
