import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import maintain as m


class MaintenanceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.root = self.base / 'provider'
        shutil.copytree(m.ROOT, self.root,
                        ignore=shutil.ignore_patterns('.git', '.local', 'bin', '__pycache__'))
        self.repo = self.base / 'upstream'
        shutil.copytree(m.ROOT / 'tools/schema/discover/fixtures', self.repo)
        for root in (self.root, self.repo):
            self.git(root, 'init', '-q')
            self.commit(root)
        self.output = self.base / 'output'
        self.calls = []

    def git(self, root, *args):
        return subprocess.check_output(['git', *args], cwd=root, stderr=subprocess.DEVNULL).decode().strip()

    def commit(self, root):
        self.git(root, 'add', '.')
        self.git(root, '-c', 'user.name=Tests', '-c', 'user.email=tests@example.invalid',
                 'commit', '-qm', 'fixture')

    def produce(self, root, archive, candidate, manifest, snapshot, destination, log):
        self.calls.append(candidate['key'])
        self.assertTrue(archive.is_file())
        self.assertEqual(m.discovery.digest((snapshot / candidate['upstream_sha'] / candidate['schema_path']).read_bytes()),
                         candidate['schema_sha256'])
        destination.mkdir(parents=True)
        (destination / 'artifact.json').write_bytes(m.discovery.encode({'key': candidate['key']}))

    def reconcile(self, **kwargs):
        return m.reconcile(self.root, self.output, repository=self.repo,
                           producer=kwargs.pop('producer', self.produce), **kwargs)

    def state(self):
        return m.discovery.load_state(self.output / 'checkpoints.json')

    def test_change_noop_and_no_live_or_publication_receipts(self):
        before = self.git(self.root, 'status', '--porcelain')
        first = self.reconcile()
        self.assertEqual(first['outcome'], 'changed')
        self.assertEqual(len(self.calls), 2)
        state = self.state()
        for entry in state['candidates'].values():
            self.assertEqual(set(entry['stages']), {'discovered', 'generated'})
        self.assertTrue(all(not c['live_tested'] and not c['published'] for c in first['candidates']))
        second = self.reconcile()
        self.assertEqual(second['outcome'], 'unchanged')
        self.assertEqual(len(self.calls), 2)
        self.assertEqual(before, self.git(self.root, 'status', '--porcelain'))

    def test_force_revalidates_without_replacing_successful_evidence(self):
        self.reconcile()
        before = self.state()['candidates']
        proof_hashes = m.inventory(self.output / 'candidates')
        self.reconcile(force=True)
        self.assertEqual(len(self.calls), 4)
        self.assertEqual(before, self.state()['candidates'])
        self.assertEqual(proof_hashes, m.inventory(self.output / 'candidates'))

    def test_source_directory_cannot_be_an_output(self):
        self.output = self.root / 'schemas'
        with self.assertRaisesRegex(m.MaintenanceError, 'outside the source tree'):
            self.reconcile()
        self.assertEqual(self.git(self.root, 'status', '--porcelain'), '')

    def test_dirty_checkout_rejected(self):
        (self.root / 'uncommitted').write_text('no')
        with self.assertRaisesRegex(m.MaintenanceError, 'clean committed'):
            self.reconcile()
        self.assertFalse(self.output.exists())

    def test_build_failure_does_not_advance_or_overwrite_baseline(self):
        self.reconcile()
        before = m.inventory(self.output / 'candidates')
        path = self.repo / 'docs/7.24.5/openapi.json'
        doc = json.loads(path.read_text())
        doc['info']['description'] = 'a real changed publication'
        path.write_text(json.dumps(doc))
        self.commit(self.repo)
        def fail(*args):
            raise m.MaintenanceError('failure')
        with self.assertRaises(m.MaintenanceError):
            self.reconcile(producer=fail)
        self.assertEqual(before, m.inventory(self.output / 'candidates'))
        self.assertEqual(json.loads((self.output / 'latest-summary.json').read_text())['outcome'], 'failure')
        self.assertEqual(sum('generated' in e['stages'] for e in self.state()['candidates'].values()), 2)
        self.reconcile()
        self.assertEqual(sum('generated' in e['stages'] for e in self.state()['candidates'].values()), 3)

    def test_review_candidate_keeps_actionable_report_without_receipt(self):
        def review(root, archive, candidate, manifest, snapshot, destination, log):
            report = {'format': 'routeros-contract-delta@1', 'decision': 'review-required',
                      'findings': [{'path': '/resources/routeros_ip_address/fields/address/type',
                                    'kind': 'value-change'}]}
            log.with_suffix('.contract-delta.json').write_bytes(m.discovery.encode(report))
            raise m.MaintenanceError('contract delta requires review')
        with self.assertRaises(m.MaintenanceError):
            self.reconcile(producer=review)
        summary = json.loads((self.output / 'latest-summary.json').read_text())
        item = summary['candidates'][0]
        self.assertEqual(item['status'], 'review-required')
        self.assertTrue((self.output / item['review_artifact']).is_file())
        self.assertTrue(all(set(e['stages']) == {'discovered'} for e in self.state()['candidates'].values()))
        self.reconcile()
        self.assertEqual(sum('generated' in e['stages'] for e in self.state()['candidates'].values()), 2)

    def test_review_failed_force_preserves_successful_receipts(self):
        self.reconcile()
        before = m.inventory(self.output / 'candidates')
        stages = self.state()['candidates']
        def review(root, archive, candidate, manifest, snapshot, destination, log):
            log.with_suffix('.contract-delta.json').write_bytes(m.discovery.encode({'decision': 'review-required'}))
            raise m.MaintenanceError('review')
        with self.assertRaises(m.MaintenanceError):
            self.reconcile(force=True, producer=review)
        self.assertEqual(before, m.inventory(self.output / 'candidates'))
        self.assertEqual(stages, self.state()['candidates'])
        self.assertEqual(self.reconcile()['outcome'], 'unchanged')

    def test_reviewed_policy_changes_invalidate_candidate_and_verification_keys(self):
        self.reconcile()
        first = set(self.calls)
        verification = m.verification_inputs(self.root)
        path = self.root / 'schemas/maintenance-policy.json'
        value = json.loads(path.read_text())
        value['revision'] = 'reviewed-next-revision'
        path.write_text(json.dumps(value))
        self.commit(self.root)
        self.assertNotEqual(verification, m.verification_inputs(self.root))
        self.reconcile()
        self.assertEqual(len(self.calls), 4)
        self.assertFalse(first & set(self.calls[2:]))

    def test_real_producer_delta_gate_stops_before_tests_or_binary(self):
        archive = self.base / 'source.tar'
        self.git(self.root, 'archive', '--format=tar', '--output=' + str(archive), 'HEAD')
        log = self.base / 'candidate.log'
        destination = self.base / 'candidate'
        original = m.run
        calls = []
        def command(args, work, log, **kwargs):
            calls.append(args)
            if args[0] == 'bash':
                path = work / 'schemas/wire-descriptors.json'
                value = json.loads(path.read_text())
                value['resources'][0]['fields'][0]['type'] = 'integer'
                path.write_bytes(m.discovery.encode(value))
            elif 'capabilities.py' in args[1]:
                original(args, work, log, **kwargs)
            else:
                self.fail('tests/build must not run after rejected delta')
        candidate = {'key': 'fixture', 'upstream_sha': 'fixture', 'schema_path': 'schema.json'}
        with patch.object(m, 'run', command), self.assertRaises(m.MaintenanceError):
            m.produce(self.root, archive, candidate, self.base / 'manifest.json', self.base,
                      destination, log)
        report = json.loads(log.with_suffix('.contract-delta.json').read_text())
        self.assertEqual(report['decision'], 'review-required')
        self.assertFalse(destination.exists())
        self.assertEqual(len(calls), 3)

    def test_corrupt_artifact_or_evidence_cannot_be_noop(self):
        self.reconcile()
        key = self.calls[0]
        artifact = self.output / 'candidates' / key / 'artifact.json'
        artifact.write_text('tampered')
        with self.assertRaisesRegex(m.MaintenanceError, 'missing or modified'):
            self.reconcile()
        artifact.unlink()
        with self.assertRaises(m.MaintenanceError):
            self.reconcile()

    def test_missing_receipted_bundle_requires_restore_not_silent_regeneration(self):
        self.reconcile()
        shutil.rmtree(self.output / 'candidates' / self.calls[0])
        with self.assertRaisesRegex(m.MaintenanceError, 'restore its bundle'):
            self.reconcile()

    def test_unreceipted_partial_bundle_is_recovered(self):
        def fail(root, archive, candidate, manifest, snapshot, destination, log):
            orphan = self.output / 'candidates' / candidate['key']
            orphan.mkdir(parents=True)
            (orphan / 'partial').write_text('not a successful artifact')
            raise m.MaintenanceError('crashed before checkpoint')
        with self.assertRaises(m.MaintenanceError):
            self.reconcile(producer=fail)
        self.reconcile()
        self.assertFalse(any((self.output / 'candidates').rglob('partial')))
        self.assertEqual(len(self.calls), 2)

    def test_missing_upstream_artifact_is_pending_not_unchanged(self):
        (self.repo / 'docs/7.24.5/openapi.json').unlink()
        self.commit(self.repo)
        summary = self.reconcile()
        self.assertEqual(summary['outcome'], 'pending')
        self.assertTrue(summary['deferred'])
        self.assertEqual(len(self.calls), 1)

    def test_invalid_discovery_preserves_checkpoints_and_last_good_bundles(self):
        self.reconcile()
        before = (self.output / 'checkpoints.json').read_bytes()
        bundles = m.inventory(self.output / 'candidates')
        (self.repo / 'docs/docs-index.json').write_text('{}')
        self.commit(self.repo)
        with self.assertRaises(m.MaintenanceError):
            self.reconcile()
        self.assertEqual(before, (self.output / 'checkpoints.json').read_bytes())
        self.assertEqual(bundles, m.inventory(self.output / 'candidates'))

    def test_producer_and_test_changes_invalidate_reuse(self):
        self.reconcile()
        path = self.root / 'internal/provider/runtime_hardening_test.go'
        path.write_text(path.read_text() + '\n// new regression test\n')
        self.commit(self.root)
        self.reconcile()
        self.assertEqual(len(self.calls), 4)
        self.assertEqual(len(self.state()['candidates']), 4)

    def test_tool_documentation_change_does_not_invalidate_offline_receipts(self):
        self.reconcile()
        doc = self.root / 'tools/contracts/README.md'
        doc.parent.mkdir(parents=True, exist_ok=True)
        doc.write_text('Documentation-only update.\n')
        self.commit(self.root)
        self.assertEqual(self.reconcile()['outcome'], 'unchanged')
        self.assertEqual(len(self.calls), 2)

    def test_ignored_tool_cache_does_not_invalidate_keys(self):
        self.reconcile()
        cache = self.root / 'tools/bin/cache.json'
        cache.parent.mkdir(parents=True, exist_ok=True)
        cache.write_text('{"ephemeral":"not a producer input"}')
        self.assertEqual(self.reconcile()['outcome'], 'unchanged')
        self.assertEqual(len(self.calls), 2)

    def test_lock_rejects_parallel_run(self):
        with m.lock(self.output):
            with self.assertRaisesRegex(m.MaintenanceError, 'another maintenance run'):
                self.reconcile()

    def test_offline_commands_strip_live_flags_and_credentials(self):
        def fake(command, **kwargs):
            env = kwargs['env']
            self.assertFalse(any(k.startswith(('ROS_', 'MIKROTIK_')) for k in env))
            self.assertNotIn('TF_ACC', env)
            self.assertNotIn('GITHUB_TOKEN', env)
            self.assertEqual(env['GOTOOLCHAIN'], 'go1.25.8')
            self.assertEqual(env['TF_ACC_TERRAFORM_VERSION'], '1.14.0')
            return subprocess.CompletedProcess(command, 0)
        with patch.dict(m.os.environ, {'ROS_PASSWORD': 'private', 'MIKROTIK_USERNAME': 'private',
                                       'TF_ACC': '1', 'GITHUB_TOKEN': 'private'}), patch.object(m.subprocess, 'run', fake):
            m.run(['go', 'test', './...'], self.root, self.base / 'offline.log')


if __name__ == '__main__':
    unittest.main()
