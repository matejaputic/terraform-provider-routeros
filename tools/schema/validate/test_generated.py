# SPDX-License-Identifier: MPL-2.0
import tempfile
import unittest
from pathlib import Path

from generated import APPROVED_PATHS, verify


class GeneratedOutputTests(unittest.TestCase):
    def test_exact_set_not_count(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            for name in APPROVED_PATHS:
                (directory / (name + "_resource_gen.go")).write_text(
                    "// official output\n"
                )
            verify(directory)
            removed = directory / "ip_dhcp_client_option_resource_gen.go"
            removed.unlink()
            with self.assertRaises(ValueError):
                verify(directory)
            extra = directory / "unreviewed_resource_gen.go"
            extra.write_text("// same count, wrong contract\n")
            with self.assertRaises(ValueError):
                verify(directory)
            removed.write_text("// restored\n")
            with self.assertRaises(ValueError):
                verify(directory)
            extra.unlink()
            (directory / "unexpected.txt").write_text("unexpected")
            with self.assertRaises(ValueError):
                verify(directory)


if __name__ == "__main__":
    unittest.main()
