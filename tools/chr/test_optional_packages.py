import hashlib
import io
import json
import tempfile
import unittest
import zipfile
from pathlib import Path
from unittest.mock import patch

import chr as harness


class OptionalPackageTests(unittest.TestCase):
    def test_recipes_match_previously_reviewed_official_archives(self):
        recipes = json.loads(
            (harness.ROOT / "tools/chr/optional-package-recipes.json").read_text()
        )
        original = json.loads(
            (harness.ROOT / "tools/chr/container-recipes.json").read_text()
        )
        self.assertEqual(set(recipes), set(original))
        for version, recipe in recipes.items():
            self.assertEqual(recipe["url"], original[version]["url"])
            self.assertEqual(
                recipe["archive_sha256"], original[version]["archive_sha256"]
            )
            self.assertEqual(
                set(recipe["packages"]), {"container", "wireless", "user-manager"}
            )
            for name, entry in recipe["packages"].items():
                self.assertEqual(entry["member"], f"{name}-{version}.npk")
                self.assertGreater(entry["package_bytes"], 0)
                self.assertEqual(len(entry["package_sha256"]), 64)
            for field in ("member", "package_bytes", "package_sha256"):
                self.assertEqual(
                    recipe["packages"]["container"][field], original[version][field]
                )

    def test_unreviewed_selection_rejected_before_any_engine_or_network(self):
        with (
            patch.object(harness, "execution_platform") as engine,
            patch.object(harness.urllib.request, "urlopen") as download,
        ):
            with self.assertRaisesRegex(RuntimeError, "Unreviewed"):
                harness.start(packages=("unreviewed",))
            with self.assertRaisesRegex(RuntimeError, "Unreviewed"):
                harness.install_container({}, harness.VERSION, ("unreviewed",))
            engine.assert_not_called()
            download.assert_not_called()

    def test_archive_and_member_hash_failure_clean_provisioning_files_before_upload(
        self,
    ):
        for corrupt_archive in (True, False):
            with (
                self.subTest(corrupt_archive=corrupt_archive),
                tempfile.TemporaryDirectory() as directory,
            ):
                root = Path(directory)
                state = root / "state"
                state.mkdir()
                recipes = root / "tools/chr/optional-package-recipes.json"
                recipes.parent.mkdir(parents=True)
                archive = io.BytesIO()
                with zipfile.ZipFile(archive, "w") as z:
                    z.writestr(f"container-{harness.VERSION}.npk", b"correct")
                    z.writestr(f"wireless-{harness.VERSION}.npk", b"tampered")
                data = archive.getvalue()
                recipe = {
                    "url": "https://invalid.test/pinned.zip",
                    "archive_sha256": "0" * 64
                    if corrupt_archive
                    else hashlib.sha256(data).hexdigest(),
                    "packages": {},
                }
                for name in ("container", "wireless"):
                    recipe["packages"][name] = {
                        "member": f"{name}-{harness.VERSION}.npk",
                        "package_bytes": 7,
                        "package_sha256": hashlib.sha256(b"correct").hexdigest(),
                    }
                    (state / f"{name}-{harness.VERSION}.npk").write_bytes(b"stale")
                recipes.write_text(json.dumps({harness.VERSION: recipe}))
                with (
                    patch.object(harness, "ROOT", root),
                    patch.object(harness, "STATE", state),
                    patch.object(
                        harness.urllib.request, "urlopen", return_value=io.BytesIO(data)
                    ),
                    patch.object(harness, "Console") as console,
                    patch.object(harness.subprocess, "run") as process,
                ):
                    with self.assertRaisesRegex(RuntimeError, "hash mismatch"):
                        harness.install_container(
                            {}, harness.VERSION, ("container", "wireless")
                        )
                    console.assert_not_called()
                    process.assert_not_called()
                self.assertFalse(list(state.glob("*.npk")))


if __name__ == "__main__":
    unittest.main()
