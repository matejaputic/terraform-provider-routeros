#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Render documentation only from reviewed descriptors/catalogs; emit no provider code."""

import argparse
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
TYPES = {
    "string": "string",
    "boolean": "bool",
    "integer": "int64",
    "array": "list(string)",
}
MODES = {
    "required": "Required",
    "computed": "Computed",
    "computed_optional": "Optional + computed",
}


def load(relative):
    return json.loads((ROOT / relative).read_text())


def inline(value):
    return "`" + str(value).replace("|", "&#124;").replace("\n", " ") + "`"


def render_all():
    resources = sorted(
        load("schemas/wire-descriptors.json")["resources"], key=lambda r: r["name"]
    )
    policies = {
        p["resource_name"]: p
        for source in (
            "collections",
            "singletons",
            "extended-collections",
            "additional-settings",
        )
        for p in load(f"internal/catalog/{source}.json")
    }
    additional = {
        p["resource_name"] for p in load("internal/catalog/additional-settings.json")
    }
    notes = load("tools/docs/resource_notes.json")
    names = {r["name"] for r in resources}
    if set(notes) - names:
        raise ValueError("notes refer to unexposed resources")
    outputs = {}
    index = [
        "# Resource reference",
        "",
        f"The provider exposes **{len(resources)} canonical resources** and **zero data sources**. These pages describe the reviewed field subsets, not every RouterOS or reference-provider option.",
        "",
        "Schemas/models are produced by the official generators; lifecycle code remains maintained separately. Publication and SDK-state migration are not claimed. See [configuration](../index.md), [coverage and evidence](../development/coverage.md), and [runtime safety](../development/runtime-policy.md).",
        "",
        "| Resource | Lifecycle | Required package |",
        "| --- | --- | --- |",
    ]
    for r in resources:
        name = r["name"]
        policy = policies.get(name, {})
        singleton = r["lifecycle"] == "reviewed-singleton"
        replacement = policy.get("replacement_only", False)
        lifecycle = (
            "Settings; destroy unmanages"
            if singleton
            else "Replacement-only collection"
            if replacement
            else "Collection CRUD"
        )
        package = policy.get("required_package")
        index.append(
            f"| [{inline(r['terraform_type'])}]({name}.md) | {lifecycle} | {inline(package) if package else '—'} |"
        )
        lines = [
            f"# {r['terraform_type']}",
            "",
            "<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->",
            "",
            f"REST path: {inline(r['path'])}. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.",
            "",
            "## Lifecycle",
            "",
        ]
        if singleton:
            identity = r["path"].lstrip("/").replace("/", ".")
            lines += [
                f"Reads the existing settings object with GET; create and update POST the fixed {inline(r['path'] + '/set')} action. Identity is {inline(identity)}.",
                "",
                "**Destroy only removes Terraform management. It does not delete, reset, or restore device settings.** A failed/malformed settings read is an error, not evidence that the singleton was deleted.",
            ]
        elif replacement:
            lines += [
                "Creates with PUT and reads by internal ID; changes require replacement rather than item PATCH. Destroy deletes the owned collection item."
            ]
        else:
            lines += [
                "Creates with PUT, reads the collection filtered by internal ID, updates with PATCH, and deletes the item with DELETE. Replacement-marked fields recreate the item rather than rename it in place."
            ]
        if not singleton:
            lines += [
                "",
                "Reviewed ownership checks can reject built-in, dynamic or otherwise unmanageable objects. Import is not blanket permission to modify every row. Failed refresh preserves prior state; a known allocated ID is retained for create recovery.",
            ]
        if package:
            lines += [
                "",
                f"**Requires the enabled {inline(package)} package matching the RouterOS system version.** Package presence does not authorize unlisted fields or certify other targets.",
            ]
        lines += [
            "",
            "## Attributes",
            "",
            "| Attribute | Type | Mode | Notes |",
            "| --- | --- | --- | --- |",
        ]
        for f in r["fields"]:
            details = []
            if f.get("force_new"):
                details.append("Change requires replacement")
            if f.get("sensitive"):
                details.append("Sensitive; protect Terraform state")
            if f.get("choices"):
                details.append(
                    "Choices: " + ", ".join(inline(json.dumps(c)) for c in f["choices"])
                )
            if "minimum" in f or "maximum" in f:
                details.append(
                    "Bounds: "
                    + str(f.get("minimum", "unbounded"))
                    + " to "
                    + str(f.get("maximum", "unbounded"))
                )
            if f.get("constraint"):
                details.append("Validation: " + inline(f["constraint"]))
            if f.get("codec") == "csv-string-set":
                details.append(
                    "Comma-separated set in a Terraform string, not an HCL list"
                )
            if f.get("codec") == "routeros-duration":
                details.append("Reviewed RouterOS duration string")
            if f.get("codec") == "auto-decimal":
                details.append("Automatic server value may read as null")
            if f.get("preserve_secret_on_omission"):
                details.append("Preserves known secret on omitted/masked readback")
            if f.get("conditional_read"):
                details.append("Conditional read: " + inline(f["conditional_read"]))
            if "read_default" in f:
                details.append(
                    "Reviewed omitted read value: "
                    + inline(json.dumps(f["read_default"]))
                )
            lines.append(
                f"| {inline(f['name'])} | {TYPES[f['type']]} | {MODES[f['mode']]} | {'; '.join(details) or '—'} |"
            )
        if any(f["mode"] == "computed_optional" for f in r["fields"]):
            lines += [
                "",
                "Optional + computed fields adopt server values when omitted; omission does not reset them. Explicit empty values clear only fields whose reviewed codec/runtime supports clearing; invalid combinations still produce diagnostics.",
            ]
        lines += ["", "Computed attributes and IDs are never mutation inputs."]
        if any(f.get("sensitive") for f in r["fields"]):
            lines += [
                "",
                "Terraform sensitivity redacts normal display; it does **not** encrypt state. Use a protected state backend. Secret preservation is per-field, not a generic substitution for any masked value.",
            ]
        lines += [
            "",
            "## Import",
            "",
            "```sh",
            f"terraform import {r['terraform_type']}.example '{identity if singleton else '*A'}'",
            "```",
            "",
            (
                "Use the exact path-derived identity above, not an internal item ID."
                if singleton
                else "Use an internal `*HEX` ID, not a name or numeric row index."
            )
            + " Import does not imply compatibility with historical SDK state.",
        ]
        if name in notes:
            lines += ["", "## Resource-specific notes", ""] + [
                "- " + note for note in notes[name]
            ]
        lines += ["", "## Verification scope", ""]
        if name not in additional:
            lines += [
                "This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate."
            ]
        elif name == "system_led_settings":
            lines += [
                "Schema/mock/race coverage exists; physical LED controls are unavailable on CHR and have no physical-board live certification."
            ]
        elif package:
            lines += [
                "Focused dirty-source configuration lifecycles passed on both pinned x86_64 targets with matching container, wireless and user-manager packages. This resource is unavailable in base lanes without its required package. Consolidated clean-source/hosted and maintenance-candidate certification of these additional settings is not recorded."
            ]
        else:
            lines += [
                "Focused dirty-source configuration lifecycles passed on both pinned x86_64/base targets and matching optional-package targets. Consolidated clean-source/hosted and maintenance-candidate certification of these additional settings is not recorded."
            ]
        lines += [
            "",
            "See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.",
            "",
            "For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.",
            "",
        ]
        outputs[f"docs/resources/{name}.md"] = "\n".join(lines)
    outputs["docs/resources/index.md"] = "\n".join(index) + "\n"
    return outputs


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--check",
        action="store_true",
        help="Reject stale/missing/extra resource reference pages",
    )
    args = parser.parse_args()
    outputs = render_all()
    expected = {ROOT / name for name in outputs}
    extra = set((ROOT / "docs/resources").glob("*.md")) - expected
    if extra:
        raise SystemExit(
            "Unreviewed resource documentation pages: "
            + ", ".join(str(p) for p in sorted(extra))
        )
    stale = []
    for name, text in outputs.items():
        path = ROOT / name
        if args.check:
            if not path.exists() or path.read_text() != text:
                stale.append(name)
        else:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text)
    if stale:
        raise SystemExit("Stale resource documentation: " + ", ".join(stale))
    print(
        f"{'Verified' if args.check else 'Rendered'} {len(outputs) - 1} resource pages and their index"
    )


if __name__ == "__main__":
    main()
