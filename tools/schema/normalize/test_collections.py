# SPDX-License-Identifier: MPL-2.0
import copy
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import normalize as n
import verify


class CollectionAdaptationTests(unittest.TestCase):
    def setUp(self):
        self.raw = (
            n.ROOT / "tools/schema/normalize/fixtures/collections.upstream.json"
        ).read_bytes()
        self.policies = json.loads(
            (n.ROOT / "internal/catalog/collections.json").read_bytes()
        )

    def test_all_reviewed_collection_contracts(self):
        paths = set()
        for policy in self.policies:
            spec, config, wire, report = n.normalize(self.raw, policy)
            self.assertEqual(report["supported_resources"], [policy["resource_name"]])
            self.assertFalse(paths & set(spec["paths"]))
            paths.update(spec["paths"])
            self.assertEqual(wire["resources"][0]["path"], policy["wire_path"])
            self.assertIn(policy["resource_name"].encode(), config)
            if policy["resource_name"] == "ip_pool":
                body = spec["paths"]["/ip/pool"]["put"]["requestBody"]["content"][
                    "application/json"
                ]["schema"]
                self.assertEqual(
                    body["properties"]["ranges"],
                    {"type": "array", "items": {"type": "string"}},
                )

    def test_unreviewed_type_default_replacement_conditional_rejected(self):
        for key, value in [
            ("type", "object"),
            ("read_default", "invented"),
            ("conditional_read", "invented"),
            ("force_new", True),
        ]:
            p = copy.deepcopy(self.policies[0])
            field = next(f for f in p["attributes"] if f["name"] == "comment")
            field[key] = value
            with self.assertRaises(n.AdaptationError):
                n.normalize(self.raw, p)

    def test_collection_loss_and_list_element_drift(self):
        spec = json.loads(self.raw)
        del spec["paths"]["/ip/pool/{id}"]["patch"]
        with self.assertRaises(n.AdaptationError):
            n.normalize(
                n.discover.encode(spec),
                next(p for p in self.policies if p["resource_name"] == "ip_pool"),
            )
        p = copy.deepcopy(
            next(p for p in self.policies if p["resource_name"] == "ip_pool")
        )
        next(f for f in p["attributes"] if f["name"] == "ranges")["codec"] = (
            "unimplemented"
        )
        with self.assertRaises(n.AdaptationError):
            n.normalize(self.raw, p)

    def test_catalog_cli_bundle_and_last_good_on_failure(self):
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp) / "out"
            command = [
                sys.executable,
                str(n.ROOT / "tools/schema/normalize/normalize.py"),
                "--collections",
                "--output",
                str(output),
            ]
            result = subprocess.run(command, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr.decode())
            report = verify.verify(output)
            self.assertEqual(set(report["supported_resources"]), set(n.APPROVED_PATHS))
            self.assertEqual(
                report["coverage"]["approved_maintained_resources"],
                len(n.APPROVED_PATHS),
            )
            self.assertEqual((output / "upstream-input.json").read_bytes(), self.raw)
            before = {
                p.name: p.read_bytes()
                for p in output.iterdir()
                if p.is_file() and not p.name.endswith(".lock")
            }
            broken = Path(tmp) / "broken.json"
            spec = json.loads(self.raw)
            del spec["paths"]["/interface/vlan/{id}"]
            broken.write_bytes(n.discover.encode(spec))
            result = subprocess.run(
                command + ["--input", str(broken)], capture_output=True
            )
            self.assertNotEqual(result.returncode, 0)
            after = {
                p.name: p.read_bytes()
                for p in output.iterdir()
                if p.is_file() and not p.name.endswith(".lock")
            }
            self.assertEqual(before, after)


if __name__ == "__main__":
    unittest.main()
