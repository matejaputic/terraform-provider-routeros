# SPDX-License-Identifier: MPL-2.0
"""Immutable parsing reuse must not cache approvals or leak report mutations."""

import copy
import json
import unittest
from pathlib import Path
from unittest.mock import patch

import normalize as n

HERE = Path(__file__).resolve().parent


class SnapshotTest(unittest.TestCase):
    def setUp(self):
        self.raw = (HERE / "fixtures/collections.upstream.json").read_bytes()
        self.policy = json.loads(
            (n.ROOT / "schemas/ip-address-policy.json").read_text()
        )
        self.collections = json.loads(
            (n.ROOT / "internal/catalog/collections.json").read_text()
        )

    def test_inventory_observed_once_and_shared_reports_match_standalone(self):
        expected = n.normalize(self.raw, self.policy)
        snapshot = n.SchemaSnapshot(self.raw)
        with patch.object(n, "inventory", wraps=n.inventory) as inventory:
            actual = n.normalize(self.raw, self.policy, snapshot=snapshot)
            second = n.normalize(
                self.raw,
                self.collections[0],
                snapshot=snapshot,
                include_inventory=False,
            )
            third = n.normalize(self.raw, self.policy, snapshot=snapshot)
            self.assertEqual(inventory.call_count, 1)
        self.assertEqual(actual, expected)
        self.assertEqual(third, expected)
        self.assertEqual(second[3]["catalog"], [])
        standalone = n.normalize(self.raw, self.collections[0])
        self.assertEqual(second[:3], standalone[:3])
        self.assertEqual(second[3]["adaptations"], standalone[3]["adaptations"])
        self.assertEqual(second[3]["coverage"], standalone[3]["coverage"])

    def test_input_bytes_are_parsed_once_across_resource_policies(self):
        parse = n.discover.parse
        calls = []

        def recording_parse(raw):
            if raw == self.raw:
                calls.append(raw)
            return parse(raw)

        with patch.object(n.discover, "parse", side_effect=recording_parse):
            snapshot = n.SchemaSnapshot(self.raw)
            n.normalize(self.raw, self.policy, snapshot=snapshot)
            for policy in self.collections:
                n.normalize(
                    self.raw, policy, snapshot=snapshot, include_inventory=False
                )
        self.assertEqual(len(calls), 1)

    def test_report_reclassification_cannot_taint_cached_inventory(self):
        snapshot = n.SchemaSnapshot(self.raw)
        report = n.normalize(self.raw, self.policy, snapshot=snapshot)[3]
        entry = next(e for e in report["catalog"] if e["path"] == "/ip/address")
        self.assertEqual(entry["classification"], "approved-maintained-resource")
        entry["classification"] = "tampered"
        cached = next(
            e for e in snapshot.observed_inventory() if e["path"] == "/ip/address"
        )
        self.assertEqual(cached["classification"], "collection-candidate-unreviewed")
        again = n.normalize(self.raw, self.policy, snapshot=snapshot)[3]
        self.assertEqual(
            next(e for e in again["catalog"] if e["path"] == "/ip/address")[
                "classification"
            ],
            "approved-maintained-resource",
        )

    def test_different_input_cannot_reuse_cached_snapshot(self):
        with self.assertRaisesRegex(
            n.AdaptationError, "cached schema input hash mismatch"
        ):
            n.normalize(
                self.raw + b"\n", self.policy, snapshot=n.SchemaSnapshot(self.raw)
            )

    def test_cached_observations_do_not_authorize_new_resource_or_bad_fields(self):
        snapshot = n.SchemaSnapshot(self.raw)
        n.normalize(self.raw, self.policy, snapshot=snapshot)
        bad = copy.deepcopy(self.policy)
        bad["wire_path"] = "/user"
        with self.assertRaisesRegex(
            n.AdaptationError, "unsupported lifecycle registration"
        ):
            n.normalize(self.raw, bad, snapshot=snapshot)
        bad = copy.deepcopy(self.policy)
        bad["attributes"][0]["sensitive"] = True
        with self.assertRaisesRegex(n.AdaptationError, "sensitivity/replacement"):
            n.normalize(self.raw, bad, snapshot=snapshot)


if __name__ == "__main__":
    unittest.main()
