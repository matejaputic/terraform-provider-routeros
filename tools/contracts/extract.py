#!/usr/bin/env python3
"""Pinned, read-only SDK source contracts; AST inspection, never SDK execution."""

import argparse
import fcntl
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PIN = "0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb"
LIMIT = 64 * 1024 * 1024
FORMAT = "routeros-reference-contracts@1"


def encode(value):
    return (
        json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
        + "\n"
    ).encode()


def digest(value):
    return hashlib.sha256(value).hexdigest()


def git(repository, *args):
    return subprocess.check_output(["git", "-C", str(repository), *args], timeout=60)


def committed_files(repository, revision=PIN):
    if revision != PIN:
        raise ValueError("reference revision must match the reviewed immutable pin")
    actual = git(repository, "rev-parse", revision + "^{commit}").decode().strip()
    if actual != PIN:
        raise ValueError("reference commit identity mismatch")
    paths = (
        git(repository, "ls-tree", "-r", "--name-only", revision, "routeros/")
        .decode()
        .splitlines()
    )
    paths = [p for p in paths if p.endswith(".go")]
    if (
        not paths
        or len(paths) > 2048
        or any(not re.fullmatch(r"routeros/[A-Za-z0-9_ /-]+\.go", p) for p in paths)
    ):
        raise ValueError("unsafe or unbounded reference inventory")
    raw = subprocess.run(
        ["git", "-C", str(repository), "cat-file", "--batch"],
        input="".join(revision + ":" + p + "\n" for p in paths).encode(),
        capture_output=True,
        check=True,
        timeout=60,
    ).stdout
    if len(raw) > LIMIT:
        raise ValueError("reference source exceeds bound")
    files = {}
    offset = 0
    for p in paths:
        end = raw.index(b"\n", offset)
        header = raw[offset:end].split()
        if len(header) != 3 or header[1] != b"blob":
            raise ValueError("expected committed Go blob")
        size = int(header[2])
        body = raw[end + 1 : end + 1 + size]
        if len(body) != size or raw[end + 1 + size : end + 2 + size] != b"\n":
            raise ValueError("truncated reference blob")
        files[p] = body.decode("utf-8")
        offset = end + 2 + size
    if offset != len(raw):
        raise ValueError("unexpected trailing Git output")
    return files


def inspect(files):
    env = {
        k: v
        for k, v in os.environ.items()
        if not k.startswith(("ROS_", "MIKROTIK_", "TF_ACC"))
    }
    env["GOTOOLCHAIN"] = "go1.25.8"
    result = subprocess.run(
        ["go", "run", "tools/contracts/main.go"],
        cwd=ROOT,
        env=env,
        input=encode({"files": files}),
        capture_output=True,
        timeout=120,
        check=True,
    )
    return json.loads(result.stdout)


def reconcile(inventory, descriptors):
    rows = []
    registered = inventory["registrations"]["resources"]
    # The curated definitions authorize exposure. Frozen constructor identity is
    # only correlation metadata for intentional canonical names, not aliases.
    policies = [
        p
        for file in (
            "collections.json",
            "singletons.json",
            "extended-collections.json",
            "additional-settings.json",
        )
        for p in json.loads((ROOT / "internal/catalog" / file).read_bytes())
    ]
    identities = {
        "routeros_" + p["resource_name"]: p["reference_constructor"]
        for p in policies
        if "reference_constructor" in p
    }
    for resource in descriptors["resources"]:
        name = resource["terraform_type"]
        if resource.get("reference_constructor") != identities.get(name):
            raise ValueError("unreviewed canonical constructor identity: " + name)
        constructor = identities.get(name, registered.get(name))
        if constructor is None or constructor not in inventory["constructors"]:
            raise ValueError("approved resource absent from pinned reference")
        contract = inventory["constructors"][constructor]
        source_fields = contract["fields"]
        comparisons = []
        for field in resource["fields"]:
            key = field["name"]
            ref = source_fields.get(key)
            if ref is None:
                comparisons.append(
                    {
                        "name": key,
                        "status": "synthetic-or-unresolved-reference",
                        "current": field,
                        "requires_review": True,
                    }
                )
                continue
            props = ref.get("properties", {})
            kinds = {
                "schema.TypeString": "string",
                "schema.TypeBool": "boolean",
                "schema.TypeInt": "integer",
                "schema.TypeFloat": "number",
                "schema.TypeList": "array",
                "schema.TypeSet": "array",
                "schema.TypeMap": "object",
            }
            source_type = kinds.get(props.get("Type", {}).get("expression"))
            flags = {
                k.lower(): (
                    props[k].get("value")
                    if k in props
                    else False
                    if ref["status"] == "static-declaration"
                    else None
                )
                for k in ("Required", "Optional", "Computed", "Sensitive", "ForceNew")
            }
            differences = []
            current_type = field["type"]
            if source_type is not None and source_type != current_type:
                differences.append("type")
            modes = [k for k in ("required", "optional", "computed") if flags[k]]
            reference_mode = (
                "computed_optional"
                if modes == ["optional", "computed"]
                else "_".join(modes)
            )
            if (
                all(
                    isinstance(flags[k], bool)
                    for k in ("required", "optional", "computed")
                )
                and modes
                and reference_mode != field["mode"]
            ):
                differences.append("mode")
            if (
                flags["forcenew"] is not None
                and bool(field.get("force_new", False)) != flags["forcenew"]
            ):
                differences.append("replacement")
            if (
                flags["sensitive"] is not None
                and bool(field.get("sensitive", False)) != flags["sensitive"]
            ):
                differences.append("sensitivity")
            comparisons.append(
                {
                    "name": key,
                    "current": field,
                    "reference": ref,
                    "reference_type": source_type,
                    "reference_flags": flags,
                    "declaration_differences": differences,
                    "status": "declaration-comparison-only",
                    "requires_review": True,
                }
            )
        current_keys = {f["name"] for f in resource["fields"]}
        rows.append(
            {
                "resource": name,
                "constructor": constructor,
                "source_status": contract["status"],
                "current_field_count": len(current_keys),
                "reference_declared_field_count": len(source_fields),
                "unexposed_reference_fields": sorted(set(source_fields) - current_keys),
                "fields": comparisons,
            }
        )
    return {
        "format": "routeros-contract-reconciliation@1",
        "source_revision": PIN,
        "descriptor_sha256": digest(encode(descriptors)),
        "policy_inputs_sha256": {
            p: digest((ROOT / p).read_bytes())
            for p in (
                "schemas/ip-address-policy.json",
                "internal/catalog/collections.json",
                "internal/catalog/singletons.json",
                "internal/catalog/extended-collections.json",
                "internal/catalog/additional-settings.json",
            )
        },
        "public_schema_changes": False,
        "automatic_promotion": False,
        "scope": "static declarations only; overlays remain authoritative; differences need review",
        "resources": rows,
    }


def extract(repository):
    files = committed_files(repository)
    inventory = inspect(files)
    resources = inventory["registrations"]["resources"]
    counts = {
        "resource_names": len(resources),
        "resource_constructors": len(set(resources.values())),
        "data_source_names": len(inventory["registrations"]["data_sources"]),
        "test_scenarios": len(inventory["tests"]),
    }
    if counts["resource_names"] != 256 or counts["resource_constructors"] != 232:
        raise ValueError(
            "pinned resource/alias inventory does not match reviewed baseline"
        )
    for contract in inventory["constructors"].values():
        stem = Path(contract["source"]["file"]).stem
        contract["related_test_sources"] = sorted(
            {
                t["source"]["file"]
                for t in inventory["tests"]
                if Path(t["source"]["file"]).stem == stem + "_test"
            }
        )
        contract["semantic_status"] = (
            "needs-review; declarations never authorize exposure"
        )
    inventory.update(
        format=FORMAT,
        source={
            "repository": "terraform-routeros/terraform-provider-routeros",
            "revision": PIN,
            "license": "MPL-2.0",
            "files_sha256": {p: digest(v.encode()) for p, v in files.items()},
        },
        extractor={
            "go": "1.25.8",
            "files_sha256": {
                p: digest((ROOT / p).read_bytes())
                for p in ("tools/contracts/main.go", "tools/contracts/extract.py")
            },
        },
        counts=counts,
        scope={
            "schema_construction_executed": False,
            "sdk_callbacks_executed": False,
            "function_semantics_ported": False,
            "declarations_are_not_runtime_contracts": True,
        },
    )
    return inventory


def write_bundle(output, inventory, report):
    output.mkdir(parents=True, exist_ok=True)
    contract_bytes, report_bytes = encode(inventory), encode(report)
    manifest = {
        "format": "routeros-contract-bundle@1",
        "reference_sha": PIN,
        "files_sha256": {
            "reference-contracts.json": digest(contract_bytes),
            "reconciliation.json": digest(report_bytes),
        },
    }
    with (output / ".contracts.lock").open("a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        for name, data in (
            ("reference-contracts.json", contract_bytes),
            ("reconciliation.json", report_bytes),
            ("manifest.json", encode(manifest)),
        ):
            # Manifest last: failed/partial writes cannot pass verification.
            with tempfile.NamedTemporaryFile(dir=output, delete=False) as f:
                f.write(data)
                f.flush()
                os.fsync(f.fileno())
                temporary = f.name
            try:
                os.replace(temporary, output / name)
            finally:
                Path(temporary).unlink(missing_ok=True)
        verify_bundle(output)


def verify_bundle(output):
    manifest = json.loads((output / "manifest.json").read_text())
    if (
        manifest.get("format") != "routeros-contract-bundle@1"
        or manifest.get("reference_sha") != PIN
        or set(manifest.get("files_sha256", {}))
        != {"reference-contracts.json", "reconciliation.json"}
    ):
        raise ValueError("invalid contract bundle manifest")
    for name, expected in manifest["files_sha256"].items():
        if digest((output / name).read_bytes()) != expected:
            raise ValueError("partial or modified contract bundle")
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repository-dir", type=Path, required=True)
    parser.add_argument("--output", type=Path, default=ROOT / ".local/contracts")
    args = parser.parse_args()
    try:
        inventory = extract(args.repository_dir)
        descriptors = json.loads((ROOT / "schemas/wire-descriptors.json").read_text())
        report = reconcile(inventory, descriptors)
        write_bundle(args.output, inventory, report)
        print(json.dumps(inventory["counts"], sort_keys=True))
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        print("Contract extraction failed: " + str(error), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
