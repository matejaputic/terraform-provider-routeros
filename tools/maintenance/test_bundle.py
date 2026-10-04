import io
import shutil
import tarfile
import unittest

import bundle as b
import maintain as m
import test_maintain as fixtures


class BundleTests(fixtures.MaintenanceFixture):
    def pack(self):
        self.reconcile()
        self.archive = self.base / "bundle.tar"
        return b.export(self.output, self.archive)

    def test_fresh_runner_restore_and_verified_noop(self):
        sha = self.pack()
        old = m.inventory(self.output / "candidates")
        shutil.rmtree(self.output)
        b.restore(self.archive, self.output, sha)
        self.assertEqual(old, m.inventory(self.output / "candidates"))
        self.assertEqual(self.reconcile()["outcome"], "unchanged")
        self.assertEqual(len(self.calls), 2)
        self.assertTrue(list((self.output / "runs").glob("*/discovery/manifest.json")))

    def test_tampered_digest_and_existing_destination_rejected(self):
        sha = self.pack()
        target = self.base / "fresh"
        with self.assertRaisesRegex(m.MaintenanceError, "digest mismatch"):
            b.restore(self.archive, target, "0" * 64)
        self.assertFalse(target.exists())
        with self.assertRaisesRegex(m.MaintenanceError, "new output directory"):
            b.restore(self.archive, self.output, sha)

    def test_export_is_immutable_and_deterministic(self):
        sha = self.pack()
        another = self.base / "second.tar"
        self.assertEqual(sha, b.export(self.output, another))
        with self.assertRaises(FileExistsError):
            b.export(self.output, self.archive)
        self.assertEqual(sha, b.digest(self.archive))

    def test_missing_or_modified_receipted_artifact_cannot_export(self):
        self.pack()
        proof = self.output / "candidates" / self.calls[0] / "artifact.json"
        proof.write_text("modified")
        with self.assertRaisesRegex(m.MaintenanceError, "missing or modified"):
            b.export(self.output, self.base / "bad.tar")
        self.assertFalse((self.base / "bad.tar").exists())

    def test_symlink_raw_input_rejected(self):
        self.reconcile()
        link = next((self.output / "runs").glob("*/discovery")) / "link"
        link.symlink_to(self.root / "go.mod")
        with self.assertRaisesRegex(m.MaintenanceError, "symlinks"):
            b.export(self.output, self.base / "bad.tar")

    def test_unsafe_duplicate_link_and_oversized_members_rejected(self):
        for kind in ("traversal", "duplicate", "link", "oversized"):
            with self.subTest(kind=kind):
                archive = self.base / (kind + ".tar")
                with tarfile.open(archive, "w") as out:
                    info = tarfile.TarInfo(
                        "../escape" if kind == "traversal" else "bundle.json"
                    )
                    if kind == "link":
                        info.type = tarfile.SYMTYPE
                        info.linkname = "/etc/passwd"
                    elif kind == "oversized":
                        info.size = b.LIMIT + 1
                    if kind == "oversized":
                        out.fileobj.write(info.tobuf())
                    else:
                        out.addfile(info)
                    if kind == "duplicate":
                        out.addfile(info)
                target = self.base / ("fresh-" + kind)
                with self.assertRaises(m.MaintenanceError):
                    b.restore(archive, target, b.digest(archive))
                self.assertFalse(target.exists())
                self.assertFalse((self.base / "escape").exists())

    def test_rehashed_archive_with_modified_artifact_rejected(self):
        self.pack()
        altered = self.base / "altered.tar"
        with tarfile.open(self.archive) as source, tarfile.open(altered, "w") as out:
            for member in source:
                raw = source.extractfile(member)
                assert raw is not None
                with raw:
                    data = raw.read()
                if member.name.endswith("/artifact.json"):
                    data = b"tampered"
                    member.size = len(data)
                out.addfile(member, io.BytesIO(data))
        target = self.base / "fresh"
        with self.assertRaisesRegex(m.MaintenanceError, "inventory mismatch"):
            b.restore(altered, target, b.digest(altered))
        self.assertFalse(target.exists())

    def test_missing_or_tampered_snapshot_cannot_export(self):
        self.reconcile()
        path = next(
            (self.output / "runs").glob("*/discovery/*/docs/7.24.5/openapi.json")
        )
        path.write_text("{}")
        with self.assertRaisesRegex(
            m.MaintenanceError, "snapshot artifact hash mismatch"
        ):
            b.export(self.output, self.base / "bad.tar")
        self.assertFalse((self.base / "bad.tar").exists())

    def test_unavailable_later_stage_verifier_fails_closed(self):
        self.reconcile()
        state_path = self.output / "checkpoints.json"
        state = m.discovery.load_state(state_path)
        key = self.calls[0]
        state["candidates"][key]["stages"]["tested"] = {"evidence_sha256": "0" * 64}
        state_path.write_bytes(m.discovery.encode(state))
        with self.assertRaisesRegex(m.MaintenanceError, "tested/published"):
            b.export(self.output, self.base / "bad.tar")

    def test_changed_verification_inputs_do_not_get_blessed_by_restore(self):
        sha = self.pack()
        shutil.rmtree(self.output)
        b.restore(self.archive, self.output, sha)
        proof = m.discovery.parse(
            (
                self.output / "candidates" / self.calls[0] / "generated-evidence.json"
            ).read_bytes()
        )
        state = m.discovery.load_state(self.output / "checkpoints.json")
        with self.assertRaisesRegex(m.MaintenanceError, "verification inputs changed"):
            m.reusable(self.output, {"key": self.calls[0]}, state, "0" * 64)
        self.assertNotEqual(proof["verification_inputs_sha256"], "0" * 64)


if __name__ == "__main__":
    unittest.main()
