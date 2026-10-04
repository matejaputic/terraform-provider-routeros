# SPDX-License-Identifier: MPL-2.0
"""Public repository terminology must not depend on temporary development groups."""

import re
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


class RepositoryNamingTests(unittest.TestCase):
    def test_no_temporary_group_labels_in_repository_paths_or_content(self):
        # Assemble the forbidden prefix so the guard itself uses neutral names.
        prefix = "bat" + "ch"
        pattern = re.compile(
            prefix + r"[ _-]?[ab](?![a-z])|" + prefix + r"[AB]", re.IGNORECASE
        )
        names = (
            subprocess.check_output(
                ["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"],
                cwd=ROOT,
            )
            .decode()
            .split("\0")
        )
        findings = []
        for name in sorted(set(filter(None, names))):
            path = ROOT / name
            if not path.is_file():
                continue  # Deleted or renamed files may still be in the index.
            if pattern.search(name):
                findings.append(name + ": filename")
            try:
                text = path.read_text(encoding="utf-8")
            except UnicodeDecodeError:
                continue
            for number, line in enumerate(text.splitlines(), 1):
                if pattern.search(line):
                    findings.append(f"{name}:{number}")
        self.assertEqual(findings, [], "\n".join(findings))


if __name__ == "__main__":
    unittest.main()
