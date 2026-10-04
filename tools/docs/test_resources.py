# SPDX-License-Identifier: MPL-2.0
import re
import unittest

import resources


class ResourceDocumentationTests(unittest.TestCase):
    def test_exact_resource_pages_and_content(self):
        rendered = resources.render_all()
        actual = {
            str(p.relative_to(resources.ROOT))
            for p in (resources.ROOT / "docs/resources").glob("*.md")
        }
        self.assertEqual(actual, set(rendered))
        for name, text in rendered.items():
            with self.subTest(path=name):
                self.assertEqual((resources.ROOT / name).read_text(), text)

    def test_all_attributes_types_and_modes_match_descriptors(self):
        descriptors = resources.load("schemas/wire-descriptors.json")["resources"]
        self.assertEqual(len(descriptors), len(resources.render_all()) - 1)
        for resource in descriptors:
            text = (
                resources.ROOT / "docs/resources" / (resource["name"] + ".md")
            ).read_text()
            rows = re.findall(
                r"^\| `([^`]+)` \| ([^|]+) \| ([^|]+) \|", text, re.MULTILINE
            )
            documented = {name: (typ.strip(), mode.strip()) for name, typ, mode in rows}
            expected = {
                f["name"]: (resources.TYPES[f["type"]], resources.MODES[f["mode"]])
                for f in resource["fields"]
            }
            with self.subTest(resource=resource["name"]):
                self.assertEqual(documented, expected)
                for f in resource["fields"]:
                    if f["sensitive"]:
                        row = next(
                            line
                            for line in text.splitlines()
                            if line.startswith("| `" + f["name"] + "` |")
                        )
                        self.assertIn("Sensitive", row)

    def test_settings_import_and_unmanagement_are_explicit(self):
        for resource in resources.load("schemas/wire-descriptors.json")["resources"]:
            if resource["lifecycle"] != "reviewed-singleton":
                continue
            text = (
                resources.ROOT / "docs/resources" / (resource["name"] + ".md")
            ).read_text()
            identity = resource["path"].lstrip("/").replace("/", ".")
            with self.subTest(resource=resource["name"]):
                self.assertIn(
                    "terraform import "
                    + resource["terraform_type"]
                    + ".example '"
                    + identity
                    + "'",
                    text,
                )
                self.assertIn("Destroy only removes Terraform management", text)

    def test_relative_markdown_links_resolve(self):
        missing = []
        for path in (resources.ROOT / "docs").rglob("*.md"):
            for target in re.findall(r"\]\(([^\s)]+)\)", path.read_text()):
                if target.startswith("#") or re.match(r"^[a-zA-Z]+:", target):
                    continue
                name = target.split("#", 1)[0]
                if name and not (path.parent / name).exists():
                    missing.append(
                        str(path.relative_to(resources.ROOT)) + ": " + target
                    )
        self.assertEqual(missing, [])

    def test_current_counts_and_policy_are_consistent(self):
        descriptors = resources.load("schemas/wire-descriptors.json")["resources"]
        matrix = resources.load("schemas/capability-matrix.json")["counts"]
        self.assertEqual(len(descriptors), matrix["implemented_subsets"])
        self.assertEqual(
            len(resources.load("schemas/provenance.json")["exposed_resources"]),
            len(descriptors),
        )
        self.assertIn(
            str(len(descriptors)), (resources.ROOT / "docs/index.md").read_text()
        )
        self.assertIn(
            str(matrix["resource_constructors"] - len(descriptors)),
            (resources.ROOT / "docs/development/coverage.md").read_text(),
        )
        revision = resources.load("schemas/maintenance-policy.json")["revision"]
        self.assertIn(
            revision, (resources.ROOT / "docs/development/capabilities.md").read_text()
        )
        self.assertIn(
            revision, (resources.ROOT / "docs/development/maintenance.md").read_text()
        )


if __name__ == "__main__":
    unittest.main()
