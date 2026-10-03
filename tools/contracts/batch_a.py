#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Build a planning-only ledger from frozen IDs and immutable published inputs.

This NEVER edits a catalog, registers resources, executes reference code, crawls
RouterOS or changes implementation/test statuses to passed. Future implementation
must supply separately reviewed contracts and actual evidence for every row.
"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[2]
BASELINE = "1897fb32c03c43fb48f6ab3020cc3baf2e2b57b2"
UPSTREAM = "adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a"
REFERENCE = "0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb"
MATRIX_SHA256 = "002626579aade6fa2f0ea28d06ccc0623512bd0be90ac74b1889d7544821969b"
BASELINE_MATRIX = ROOT / "tools/contracts/fixtures/batch-a-baseline-matrix.json"
ROSTER = """ResourceCertificateScepServer
ResourceDhcpClient
ResourceDhcpRelay
ResourceDhcpServerConfig
ResourceDhcpServerOptionMatcher
ResourceDhcpServerOptionSets
ResourceDnsAdlist
ResourceDnsRecord
ResourceIPConnectionTracking
ResourceIPVrf
ResourceIPv6Address
ResourceIPv6DhcpClient
ResourceIPv6DhcpClientOption
ResourceIPv6FirewallAddrList
ResourceIPv6FirewallFilter
ResourceIPv6NeighborDiscovery
ResourceIPv6Route
ResourceInterface6to4
ResourceInterfaceBonding
ResourceInterfaceBridgeSettings
ResourceInterfaceBridgeVlan
ResourceInterfaceDot1xClient
ResourceInterfaceDot1xServer
ResourceInterfaceGre6
ResourceInterfaceL2tpClient
ResourceInterfaceMacVlan
ResourceInterfaceOpenVPNServer
ResourceInterfacePPPoEClient
ResourceInterfacePppoeServer
ResourceInterfaceSSTPClient
ResourceInterfaceVeth
ResourceInterfaceVxlan
ResourceInterfaceVxlanVteps
ResourceInterfaceWireguard
ResourceInterfaceWireguardPeer
ResourceIpDnsForwarders
ResourceIpFirewallLayer7Protocol
ResourceIpHotspot
ResourceIpHotspotIpBinding
ResourceIpHotspotProfile
ResourceIpHotspotUser
ResourceIpHotspotUserProfile
ResourceIpHotspotWalledGarden
ResourceIpHotspotWalledGardenIp
ResourceIpIpsecIdentity
ResourceIpIpsecModeConfig
ResourceIpIpsecPeer
ResourceIpIpsecPolicy
ResourceIpIpsecPolicyGroup
ResourceIpIpsecProfile
ResourceIpIpsecProposal
ResourceIpIpsecSettings
ResourceIpNeighborDiscoverySettings
ResourceIpSettings
ResourceIpTFTP
ResourceIpTFTPSettings
ResourceIpTrafficFlow
ResourceIpTrafficFlowIpfix
ResourceIpTrafficFlowTarget
ResourceIpv6DhcpServer
ResourceIpv6DhcpServerOption
ResourceIpv6DhcpServerOptionSets
ResourceIpv6NdPrefix
ResourceIpv6Pool
ResourceIpv6Settings
ResourceNatPmpInterfaces
ResourceNatPmpSettings
ResourceOpenVPNClient
ResourcePPPProfile
ResourcePPPSecret
ResourceQueueSimple
ResourceQueueTree
ResourceQueueType
ResourceRadius
ResourceRadiusIncoming
ResourceRoutingBfdConfiguration
ResourceRoutingBgpConnection
ResourceRoutingBgpEvpn
ResourceRoutingBgpInstance
ResourceRoutingBgpTemplate
ResourceRoutingBgpVpn
ResourceRoutingFilterRule
ResourceRoutingId
ResourceRoutingIgmpProxyInterface
ResourceRoutingOspfArea
ResourceRoutingOspfAreaRange
ResourceRoutingOspfInstance
ResourceRoutingOspfInterfaceTemplate
ResourceRoutingRule
ResourceSNMPCommunity
ResourceSystemIdentity
ResourceSystemLogging
ResourceSystemNote
ResourceSystemNtpClient
ResourceSystemNtpServer
ResourceSystemScheduler
ResourceToolGraphingInterface
ResourceToolGraphingQueue
ResourceToolGraphingResource
ResourceToolMacServer
ResourceToolMacServerPing
ResourceToolMacServerWinBox
ResourceToolNetwatch
ResourceUPNPInterfaces
ResourceUser
ResourceUserGroup
ResourceUserSshKeys""".splitlines()
CANONICAL = {
 "ResourceCertificateScepServer": "routeros_system_certificate_scep_server",
 "ResourceDhcpClient": "routeros_ip_dhcp_client",
 "ResourceDhcpServerOptionSets": "routeros_ip_dhcp_server_option_set",
 "ResourceDnsRecord": "routeros_ip_dns_record",
 "ResourceInterfaceBridgeVlan": "routeros_interface_bridge_vlan",
 "ResourceInterfaceWireguard": "routeros_interface_wireguard",
 "ResourceInterfaceWireguardPeer": "routeros_interface_wireguard_peer",
 "ResourceSystemIdentity": "routeros_system_identity",
 "ResourceSystemScheduler": "routeros_system_scheduler",
}
METHODS = {"get", "put", "post", "patch", "delete", "head", "options", "trace"}

def digest(raw):
 return hashlib.sha256(raw).hexdigest()

def blob(repository, revision, path):
 # Read committed text only; no checkouts, scripts or upstream Go execution.
 return subprocess.check_output(["git", "-C", str(repository), "show", f"{revision}:{path}"])

def schema_shape(spec, path):
 paths = spec["paths"]
 root = paths.get(path, {})
 item = paths.get(str(path) + "/{id}", {})
 return {
  "path_present": path in paths,
  "root_methods": sorted(METHODS & root.keys()),
  "item_methods": sorted(METHODS & item.keys()),
  "structural_collection_crud": "put" in root and {"get", "patch", "delete"} <= item.keys(),
  "authority": "observation-only; not lifecycle approval or live evidence",
 }

def build(matrix_raw, contracts_raw, snapshots):
 if digest(matrix_raw) != MATRIX_SHA256:
  raise ValueError("frozen baseline matrix hash mismatch")
 matrix = json.loads(matrix_raw)
 contracts = json.loads(contracts_raw)
 if contracts.get("source", {}).get("revision") != REFERENCE:
  raise ValueError("static reference revision mismatch")
 rows = {r["constructor"]: r for r in matrix["constructors"]}
 remaining = {n for n, r in rows.items() if not r["implemented"]}
 if len(ROSTER) != 107 or len(set(ROSTER)) != 107 or not set(ROSTER) <= remaining or len(remaining) != 214:
  raise ValueError("invalid frozen roster/remaining denominator")
 entries = []
 for name in ROSTER:
  row = rows[name]
  names = row["resource_names"]
  canonical = CANONICAL.get(name)
  if canonical is None:
   if len(names) != 1:
    raise ValueError("canonical name requires explicit selection: " + name)
   canonical = names[0]
  if canonical not in names:
   raise ValueError("canonical name is not a pinned registration")
  fields = contracts["constructors"][name].get("fields", {})
  observations = {label: schema_shape(spec, row["wire_path_candidate"]) for label, (_, _, spec) in snapshots.items()}
  notes = []
  if row["family"] == "singleton-candidate":
   notes.append("Separate GET object / POST fixed /set / path-derived ID / destroy-unmanage lifecycle. Shared helper exists unregistered; field contracts and official generation still pending.")
  if name == "ResourceInterfaceVeth":
   notes.append("Missing from pinned base schema: establish extra-package contract and lane-aware generation/runtime policy; do not fabricate base CRUD paths.")
  if name == "ResourceUserSshKeys":
   notes.append("No item PATCH observed: resolve replacement-only fields and sensitive key readback/fingerprint ownership; do not force generic update.")
  if name == "ResourceRoutingOspfInterfaceTemplate":
   notes.append("Static schema construction remains unresolved; review source/helpers rather than assume complete extracted fields.")
  if name == "ResourceDnsAdlist":
   notes.append("Acceptance must disable/isolate remote fetching; configuration CRUD is not successful fetch evidence.")
  entries.append({
   "constructor": name, "canonical_name_proposal": canonical,
   "aliases_deferred": [n for n in names if n != canonical],
   "source": row["source"], "path_candidate": row["wire_path_candidate"],
   "triage_family": row["family"], "triage_risks": row["risks"],
   "static_field_candidates": sorted(fields),
   "unresolved_field_declarations": sorted(n for n, f in fields.items() if f.get("status") != "static-declaration"),
   "schema_observations": observations, "review_notes": notes,
   "reviewed_contract": None, "implementation": "pending",
   "official_generation": "pending", "terraform_mock": "pending", "live_evidence": {},
   "automatic_exposure_authorized": False,
  })
 if len({e["canonical_name_proposal"] for e in entries}) != 107:
  raise ValueError("duplicate canonical name proposal")
 return {
  "format": "routeros-batch-a-planning@1", "scope": "planning-only; not exposure/implementation/certification authority",
  "baseline_source": BASELINE, "baseline_matrix_sha256": MATRIX_SHA256,
  "reference_sha": REFERENCE, "reference_contracts_sha256": digest(contracts_raw),
  "upstream_sha": UPSTREAM,
  "schema_inputs": {label: {"path": path, "sha256": digest(raw)} for label, (path, raw, _) in snapshots.items()},
  "batch_a_denominator": 107, "baseline_implemented": 18, "target_implemented": 125,
  "batch_b_constructor_ids": sorted(remaining - set(ROSTER)),
  "resources": entries,
 }

# CLI acquisition is separate from the pure planning transform for testing.
def main():
 parser = argparse.ArgumentParser(description=__doc__)
 parser.add_argument("--upstream-dir", required=True, type=Path)
 parser.add_argument("--output", type=Path, default=ROOT / "schemas/batch-a-ledger.json")
 args = parser.parse_args()
 snapshots = {}
 for version in ("7.24.5", "7.25beta5"):
  for flavor in ("base", "extra"):
   label = f"{version}/{flavor}"
   path = f"docs/{version}/" + ("extra/" if flavor == "extra" else "") + "openapi.json"
   raw = blob(args.upstream_dir, UPSTREAM, path)
   spec = json.loads(raw)
   if spec["info"]["version"] != version:
    raise ValueError("immutable schema version mismatch")
   snapshots[label] = (path, raw, spec)
 matrix_raw = blob(ROOT, BASELINE, "schemas/capability-matrix.json")
 if digest(matrix_raw) != MATRIX_SHA256:
  raise ValueError("frozen baseline matrix hash mismatch")
 # Archive exact historical bytes so offline tests also work in shallow CI
 # checkouts after current implementation status moves beyond eighteen.
 if BASELINE_MATRIX.exists():
  if BASELINE_MATRIX.read_bytes() != matrix_raw:
   raise ValueError("historical baseline cache changed")
 else:
  BASELINE_MATRIX.parent.mkdir(parents=True, exist_ok=True)
  BASELINE_MATRIX.write_bytes(matrix_raw)
 result = build(matrix_raw, blob(ROOT, BASELINE, "schemas/reference-contracts/reference-contracts.json"), snapshots)
 args.output.parent.mkdir(parents=True, exist_ok=True)
 data = json.dumps(result, indent=2, sort_keys=True) + "\n"
 if args.output.exists() and args.output.read_text() != data:
  raise ValueError("ledger differs: use a fresh output; never overwrite reviewed/progress evidence")
 args.output.write_text(data)
 print("107 planning rows, 107 complement IDs; zero new resources certified")

if __name__ == "__main__":
 main()
