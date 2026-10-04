import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

import chr as harness


class ExecutionPlatformTests(unittest.TestCase):
    def test_pinned_base_enabled_packages(self):
        for disabled in ("false", "no", False):
            harness.validate_packages(
                [{"name": "routeros", "version": harness.VERSION, "disabled": disabled}]
            )
        for packages in (
            None,
            [],
            ["malformed"],
            [{"name": "routeros", "version": "wrong"}],
            [{"name": "routeros", "version": harness.VERSION, "disabled": "unknown"}],
            [{"name": "routeros", "version": harness.VERSION, "disabled": 0}],
            [
                {"name": "routeros", "version": harness.VERSION},
                {"name": "extra", "version": harness.VERSION},
            ],
        ):
            with self.subTest(packages=packages), self.assertRaises(RuntimeError):
                harness.validate_packages(packages)

    def test_console_retries_proxy_socket_until_actual_serial_bytes(self):
        early = Mock()
        early.recv.side_effect = ConnectionResetError()
        ready = Mock()
        ready.recv.return_value = b"Login: "
        with (
            patch.object(
                harness.socket, "create_connection", side_effect=[early, ready]
            ),
            patch.object(harness.time, "sleep"),
        ):
            console = harness.Console()
        early.close.assert_called_once()
        self.assertIs(console.s, ready)
        self.assertEqual(console.buffer, b"Login: ")

    def test_diagnostic_markers_do_not_change_expected_prompt(self):
        console = harness.Console.__new__(harness.Console)
        console.s = Mock()
        console.s.recv.side_effect = [b"booting", b"Login: "]
        console.buffer = b""
        console.events = set()
        self.assertEqual(console.expect(b"Login:"), b"booting")
        self.assertIn("login-prompt", console.events)
        self.assertEqual(console.buffer, b" ")

    def test_unknown_recipe_refused_before_execution(self):
        with patch.object(harness, "execution_platform") as engine:
            with self.assertRaisesRegex(RuntimeError, "pinned acquisition recipe"):
                harness.start(version="unreviewed")
            engine.assert_not_called()

    def test_failed_test_cannot_retain_previous_success(self):
        with tempfile.TemporaryDirectory() as name:
            state = Path(name)
            proof = state / "acceptance-evidence.json"
            proof.write_text('{"success": true}')
            with patch.object(harness, "STATE", state), self.assertRaises(RuntimeError):
                harness.test(version="unreviewed")
            self.assertFalse(proof.exists())

    def test_local_context_preserved(self):
        with patch.object(
            harness.subprocess, "check_output", return_value="orbstack\n"
        ):
            self.assertEqual(
                harness.execution_platform(), ("linux/arm64", "OrbStack Docker")
            )

    def test_local_default_context_refused(self):
        with patch.object(harness.subprocess, "check_output", return_value="default\n"):
            with self.assertRaisesRegex(RuntimeError, "OrbStack"):
                harness.execution_platform()

    def test_owned_hosted_context_explicit_opt_in(self):
        env = {
            "GITHUB_ACTIONS": "true",
            "RUNNER_ENVIRONMENT": "github-hosted",
            "GITHUB_REPOSITORY": "matejaputic/terraform-provider-routeros",
        }
        with (
            patch.dict(os.environ, env, clear=True),
            patch.object(harness.subprocess, "check_output", return_value="default\n"),
        ):
            self.assertEqual(
                harness.execution_platform(True),
                ("linux/amd64", "GitHub-hosted Docker"),
            )
            with self.assertRaises(RuntimeError):
                harness.execution_platform()

    def test_hosted_refuses_fork_self_hosted_or_unidentified_engine(self):
        env = {
            "GITHUB_ACTIONS": "true",
            "RUNNER_ENVIRONMENT": "github-hosted",
            "GITHUB_REPOSITORY": "matejaputic/terraform-provider-routeros",
        }
        for key, value in [
            ("GITHUB_ACTIONS", "false"),
            ("RUNNER_ENVIRONMENT", "self-hosted"),
            ("GITHUB_REPOSITORY", "fork/provider"),
        ]:
            with (
                self.subTest(key=key),
                patch.dict(os.environ, {**env, key: value}, clear=True),
                patch.object(
                    harness.subprocess, "check_output", return_value="default\n"
                ),
            ):
                with self.assertRaises(RuntimeError):
                    harness.execution_platform(True)
        with (
            patch.dict(os.environ, env, clear=True),
            patch.object(harness.subprocess, "check_output", return_value="remote\n"),
        ):
            with self.assertRaises(RuntimeError):
                harness.execution_platform(True)


if __name__ == "__main__":
    unittest.main()
