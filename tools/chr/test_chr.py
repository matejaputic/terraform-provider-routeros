import os
import unittest
from unittest.mock import patch

import chr as harness


class ExecutionPlatformTests(unittest.TestCase):
    def test_local_context_preserved(self):
        with patch.object(harness.subprocess, 'check_output', return_value='orbstack\n'):
            self.assertEqual(harness.execution_platform(), ('linux/arm64', 'OrbStack Docker'))

    def test_local_default_context_refused(self):
        with patch.object(harness.subprocess, 'check_output', return_value='default\n'):
            with self.assertRaisesRegex(RuntimeError, 'OrbStack'):
                harness.execution_platform()

    def test_owned_hosted_context_explicit_opt_in(self):
        env = {'GITHUB_ACTIONS': 'true', 'RUNNER_ENVIRONMENT': 'github-hosted',
               'GITHUB_REPOSITORY': 'matejaputic/terraform-provider-routeros'}
        with patch.dict(os.environ, env, clear=True), patch.object(harness.subprocess, 'check_output', return_value='default\n'):
            self.assertEqual(harness.execution_platform(True), ('linux/amd64', 'GitHub-hosted Docker'))
            with self.assertRaises(RuntimeError):
                harness.execution_platform()

    def test_hosted_refuses_fork_self_hosted_or_unidentified_engine(self):
        env = {'GITHUB_ACTIONS': 'true', 'RUNNER_ENVIRONMENT': 'github-hosted',
               'GITHUB_REPOSITORY': 'matejaputic/terraform-provider-routeros'}
        for key, value in [('GITHUB_ACTIONS', 'false'), ('RUNNER_ENVIRONMENT', 'self-hosted'),
                           ('GITHUB_REPOSITORY', 'fork/provider')]:
            with self.subTest(key=key), patch.dict(os.environ, {**env, key: value}, clear=True), patch.object(harness.subprocess, 'check_output', return_value='default\n'):
                with self.assertRaises(RuntimeError):
                    harness.execution_platform(True)
        with patch.dict(os.environ, env, clear=True), patch.object(harness.subprocess, 'check_output', return_value='remote\n'):
            with self.assertRaises(RuntimeError):
                harness.execution_platform(True)


if __name__ == '__main__':
    unittest.main()
