import copy
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import extract as e

FIXTURE = """package routeros
const KeyName = "name"
const Prefix = "routeros_"
var PropName = &schema.Schema{Type: schema.TypeString, Required: true, ForceNew: true}
var PropCycle = PropCycle
func ResourceOne() *schema.Resource {
 s := map[string]*schema.Schema{
  KeyName: PropName,
  "enabled": {Type: schema.TypeBool, Optional: true, Computed: true, Sensitive: true, Default: false},
  "nested": {Type: schema.TypeList, Optional: true, Elem: &schema.Resource{Schema: map[string]*schema.Schema{"child": {Type: schema.TypeInt, Required: true}}}},
  "cycle": PropCycle,
  "duration": {Type: schema.TypeString, Optional: true, DiffSuppressFunc: TimeEqual, ValidateFunc: validation.StringInSlice([]string{"one", "two"}, false)},
  MetaResourcePath: PropResourcePath("/interface/list"),
 }
 return &schema.Resource{Schema: s, CreateContext: DefaultCreate(s), SchemaVersion: 2}
}
const MetaResourcePath = "___path___"
func Provider() *schema.Provider {return &schema.Provider{
 ResourcesMap: map[string]*schema.Resource{Prefix+"one": ResourceOne(), "routeros_alias": ResourceOne()},
 DataSourcesMap: map[string]*schema.Resource{"routeros_data": ResourceOne()},
}}
"""


class ExtractionTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.inventory = e.inspect(
            {
                "routeros/resource.go": FIXTURE,
                "routeros/resource_test.go": "package routeros\nfunc TestOne(t *testing.T) {}",
            }
        )

    def test_registrations_aliases_and_tests(self):
        x = self.inventory
        self.assertEqual(
            x["registrations"]["resources"],
            {"routeros_one": "ResourceOne", "routeros_alias": "ResourceOne"},
        )
        self.assertEqual(
            x["alias_groups"]["resources:ResourceOne"],
            ["routeros_alias", "routeros_one"],
        )
        self.assertEqual(x["tests"][0]["name"], "TestOne")
        self.assertEqual(x["tests"][0]["source"]["file"], "routeros/resource_test.go")

    def test_literals_helper_expansion_nested_no_flattening(self):
        c = self.inventory["constructors"]["ResourceOne"]
        self.assertEqual(c["status"], "static-declarations-only")
        self.assertEqual(c["fields"]["name"]["properties"]["Required"]["value"], True)
        self.assertEqual(c["fields"]["name"]["definition_source"]["line"], 4)
        self.assertNotIn("child", c["fields"])
        self.assertIn(
            "child", c["fields"]["nested"]["properties"]["Elem"]["expression"]
        )
        self.assertEqual(c["resource_properties"]["SchemaVersion"]["value"], 2)
        self.assertIn("___path___", c["metadata"])
        self.assertNotIn("___path___", c["fields"])

    def test_complex_semantics_and_cycles_not_evaluated(self):
        c = self.inventory["constructors"]["ResourceOne"]
        self.assertEqual(c["fields"]["cycle"]["status"], "needs-review")
        self.assertEqual(
            c["fields"]["duration"]["properties"]["DiffSuppressFunc"]["status"],
            "needs-review",
        )
        self.assertEqual(
            c["fields"]["enabled"]["properties"]["Default"]["value"], False
        )
        validator = c["fields"]["duration"]["properties"]["ValidateFunc"]
        self.assertEqual(validator["status"], "needs-review")
        self.assertFalse(validator["call"]["executed"])
        self.assertEqual(validator["call"]["arguments"][0]["value"], ["one", "two"])
        self.assertFalse("body" in self.inventory["functions"]["ResourceOne"])

    def test_mutations_and_unresolved_keys_are_reported(self):
        src = FIXTURE.replace(
            "return &schema.Resource{Schema: s,",
            's["new"] = PropName; delete(s, "enabled"); return &schema.Resource{Schema: s,',
        )
        x = e.inspect({"routeros/resource.go": src})["constructors"]["ResourceOne"]
        self.assertEqual(x["status"], "needs-review")
        self.assertEqual(
            {f["kind"] for f in x["review_findings"]},
            {"indexed-assignment", "possible-schema-mutation"},
        )
        src = FIXTURE.replace("KeyName: PropName", "DynamicKey(): PropName")
        x = e.inspect({"routeros/resource.go": src})["constructors"]["ResourceOne"]
        self.assertEqual(x["status"], "needs-review")

    def test_duplicate_and_unsupported_registration_fail(self):
        import subprocess

        for src in (
            FIXTURE.replace('"routeros_alias"', '"routeros_one"'),
            FIXTURE.replace(
                '"routeros_alias": ResourceOne()', '"routeros_alias": Factory(true)'
            ),
            FIXTURE.replace(
                '"routeros_alias": ResourceOne()', '"routeros_alias": Missing()'
            ),
        ):
            with self.assertRaises(subprocess.CalledProcessError):
                e.inspect({"routeros/resource.go": src})

    def test_determinism_independent_of_file_order(self):
        a = {"routeros/resource.go": FIXTURE, "routeros/empty.go": "package routeros"}
        self.assertEqual(
            e.encode(e.inspect(a)), e.encode(e.inspect(dict(reversed(list(a.items())))))
        )

    def test_reconciliation_is_evidence_not_promotion(self):
        descriptor = {
            "resources": [
                {
                    "terraform_type": "routeros_one",
                    "fields": [
                        {
                            "name": "name",
                            "type": "string",
                            "mode": "required",
                            "force_new": False,
                            "sensitive": False,
                        },
                        {
                            "name": "enabled",
                            "type": "string",
                            "mode": "computed",
                            "force_new": False,
                            "sensitive": False,
                        },
                        {"name": "id", "type": "string", "mode": "computed"},
                    ],
                }
            ]
        }
        report = e.reconcile(self.inventory, descriptor)
        self.assertFalse(report["public_schema_changes"])
        self.assertFalse(report["automatic_promotion"])
        row = report["resources"][0]
        self.assertEqual(row["fields"][0]["declaration_differences"], ["replacement"])
        self.assertEqual(
            row["fields"][1]["declaration_differences"], ["type", "mode", "sensitivity"]
        )
        self.assertEqual(
            row["fields"][2]["status"], "synthetic-or-unresolved-reference"
        )
        self.assertTrue(all(f["requires_review"] for f in row["fields"]))
        self.assertIn("duration", row["unexposed_reference_fields"])
        self.assertEqual(report["descriptor_sha256"], e.digest(e.encode(descriptor)))

    def test_unresolved_flags_are_unknown_not_defaults(self):
        d = {
            "resources": [
                {
                    "terraform_type": "routeros_one",
                    "fields": [
                        {
                            "name": "cycle",
                            "type": "string",
                            "mode": "required",
                            "force_new": True,
                        }
                    ],
                }
            ]
        }
        f = e.reconcile(self.inventory, d)["resources"][0]["fields"][0]
        self.assertEqual(set(f["reference_flags"].values()), {None})
        self.assertEqual(f["declaration_differences"], [])

    def test_partially_unknown_modes_are_not_inferred(self):
        inventory = copy.deepcopy(self.inventory)
        prop = inventory["constructors"]["ResourceOne"]["fields"]["enabled"][
            "properties"
        ]["Optional"]
        prop.pop("value")
        prop["status"] = "needs-review"
        d = {
            "resources": [
                {
                    "terraform_type": "routeros_one",
                    "fields": [
                        {
                            "name": "enabled",
                            "type": "boolean",
                            "mode": "required",
                            "sensitive": True,
                        }
                    ],
                }
            ]
        }
        f = e.reconcile(inventory, d)["resources"][0]["fields"][0]
        self.assertNotIn("mode", f["declaration_differences"])
        self.assertIsNone(f["reference_flags"]["optional"])

    def test_unregistered_overlay_fails_reconciliation(self):
        with self.assertRaises(ValueError):
            e.reconcile(
                self.inventory,
                {"resources": [{"terraform_type": "routeros_unknown", "fields": []}]},
            )

    def test_hash_bound_bundle_and_tampering(self):
        with tempfile.TemporaryDirectory() as name:
            output = Path(name)
            e.write_bundle(output, self.inventory, {"public_schema_changes": False})
            self.assertEqual(e.verify_bundle(output)["reference_sha"], e.PIN)
            (output / "reconciliation.json").write_text("{}")
            with self.assertRaisesRegex(ValueError, "partial or modified"):
                e.verify_bundle(output)

    def test_partial_write_cannot_validate_as_last_good_bundle(self):
        with tempfile.TemporaryDirectory() as name:
            output = Path(name)
            e.write_bundle(output, self.inventory, {"first": True})
            changed = copy.deepcopy(self.inventory)
            changed["changed"] = True
            replace = e.os.replace

            def fail(src, destination):
                if destination.name == "manifest.json":
                    raise OSError("interrupted before final marker")
                return replace(src, destination)

            with patch.object(e.os, "replace", fail), self.assertRaises(OSError):
                e.write_bundle(output, changed, {"second": True})
            with self.assertRaises(ValueError):
                e.verify_bundle(output)
            e.write_bundle(output, self.inventory, {"first": True})
            e.verify_bundle(output)

    def test_pin_cannot_be_overridden(self):
        with self.assertRaisesRegex(ValueError, "reviewed immutable pin"):
            e.committed_files(Path("/unused"), revision="a" * 40)

    def test_reads_committed_blobs_not_dirty_working_files(self):
        # The full-SHA Git object protocol is exercised without needing upstream in CI.
        body = b"package routeros\n"
        batch = b"a" * 40 + b" blob " + str(len(body)).encode() + b"\n" + body + b"\n"

        class Response:
            stdout = batch

        def git(repo, *args):
            if args[0] == "rev-parse":
                return (e.PIN + "\n").encode()
            return b"routeros/resource _test.go\n"

        with (
            patch.object(e, "git", git),
            patch.object(e.subprocess, "run", return_value=Response()) as run,
        ):
            self.assertEqual(
                e.committed_files("/unused"),
                {"routeros/resource _test.go": body.decode()},
            )
            self.assertIn(e.PIN.encode(), run.call_args.kwargs["input"])
            self.assertNotIn("checkout", run.call_args.args[0])


if __name__ == "__main__":
    unittest.main()
