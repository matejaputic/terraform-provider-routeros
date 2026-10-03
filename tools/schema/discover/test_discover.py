from datetime import datetime, timezone
from pathlib import Path
import tempfile
import shutil
import subprocess
import sys
import unittest
from unittest.mock import patch
import urllib.error

import discover as d

HERE = Path(__file__).parent
NOW = datetime(2026, 10, 2, tzinfo=timezone.utc)
SHA = 'a' * 40
INPUTS = {'policy': 'v2', 'runtime': 'v2', 'tools': {'openapi': '0.3.0', 'framework': '0.4.1'}}

class Source:
    def __init__(self):
        self.blobs = {p.relative_to(HERE / 'fixtures').as_posix(): p.read_bytes()
                      for p in (HERE / 'fixtures').rglob('*.json')}
        self.calls = []
        self.resolves = 0
    def resolve(self, requested=None):
        self.resolves += 1
        return requested or SHA
    def get(self, sha, path):
        self.calls.append((sha, path))
        if path not in self.blobs:
            raise d.Missing()
        return self.blobs[path]
    def update(self, path, change):
        obj = d.parse(self.blobs[path])
        change(obj)
        self.blobs[path] = d.encode(obj)

class DiscoveryTests(unittest.TestCase):
    def setUp(self):
        self.source = Source()
        self.state = {'format': d.FORMAT, 'candidates': {}, 'lanes': {}}
    def run_discovery(self, **kwargs):
        return d.discover(self.source, INPUTS, self.state, now=NOW, **kwargs)[0]
    def test_version_order_and_allowlist(self):
        values = ['7.24.5', '7.25_ab434', '7.25beta5', '7.25rc1', '7.25', '7.25.1']
        self.assertEqual(sorted(values, key=d.order), values)
        for bad in ['7.25; touch x', '../7.24.5', '7.025', '7.25_ab0', '8.1', '7.25beta', '7.25beta05', '7.25$(x)', None]:
            with self.subTest(bad=bad), self.assertRaises(d.DiscoveryError):
                d.version(bad)
        for bad in ['docs/7.24.5/../openapi.json', 'docs/7.24.5/extra/x/openapi.json', '/docs/7.24.5/openapi.json', 'docs/7.24.5/run.sh']:
            with self.assertRaises(d.DiscoveryError):
                d.valid_path(bad)
    def test_fixed_snapshot_and_bounded_inventory(self):
        report = self.run_discovery(extras=True, nightly=True, backfill=1)
        self.assertEqual(len(report['candidates']), 8)
        self.assertEqual(self.source.resolves, 1)
        self.assertTrue(all(sha == SHA for sha, _ in self.source.calls))
        self.assertEqual(len(self.source.calls), len(set(self.source.calls)))
        self.assertTrue(all(c['architecture'] == 'x86' for c in report['candidates']))
        self.assertEqual(report['candidates'][0]['key'], self.run_discovery(extras=True, nightly=True, backfill=1)['candidates'][0]['key'])
    def test_missing_extra_deferred_and_missing_base_visible(self):
        del self.source.blobs['docs/7.24.5/extra/openapi.json']
        report = self.run_discovery(extras=True)
        self.assertEqual(len(report['candidates']), 3)
        self.assertEqual(report['deferred'], [{'path': 'docs/7.24.5/extra/openapi.json', 'reason': 'not-yet-published'}])
        del self.source.blobs['docs/7.24.5/openapi.json']
        self.assertEqual(len(self.run_discovery()['deferred']), 1)
    def test_missing_index_is_failure_not_noop(self):
        del self.source.blobs['docs/docs-index.json']
        with self.assertRaises(d.DiscoveryError): self.run_discovery()
    def test_corrupt_selected_specs_fail(self):
        for body in [b'{', b'{}', b'{"info":{},"info":{}}', b'{"n":NaN}']:
            with self.subTest(body=body):
                self.source.blobs['docs/7.24.5/openapi.json'] = body
                with self.assertRaises(d.DiscoveryError): self.run_discovery()
    def test_version_mismatch_and_unsafe_index(self):
        self.source.update('docs/7.24.5/openapi.json', lambda x: x['info'].update(version='7.24.4'))
        with self.assertRaises(d.DiscoveryError): self.run_discovery()
        self.source = Source()
        self.source.update('docs/docs-index.json', lambda x: x['versions'][0].update(path='docs/../../escape'))
        with self.assertRaises(d.DiscoveryError): self.run_discovery()
    def test_nightly_all_architecture_versions_checked_even_without_extras(self):
        self.source.update('docs/nightly/nightly.json', lambda x: x['provenance']['arm64']['extra'].update(postVersion='7.25_ab433'))
        with self.assertRaisesRegex(d.DiscoveryError, 'mixed'): self.run_discovery(nightly=True)
        with self.assertRaisesRegex(d.DiscoveryError, 'mixed'): self.run_discovery(nightly=True, replay=True)
    def test_nightly_staleness_replay_and_force(self):
        def stale(x):
            x['builtAt'] = '2026-08-13T23:00:00Z'
            for phases in x['provenance'].values():
                for phase in phases.values(): phase['builtAt'] = '2026-08-13T22:00:00Z'
        self.source.update('docs/nightly/nightly.json', stale)
        self.source.update('docs/docs-index.json', lambda x: x['nightly'].update(builtAt='2026-08-13T23:00:00Z'))
        for kwargs in [{}, {'force': True}]:
            with self.assertRaisesRegex(d.DiscoveryError, 'stale'): self.run_discovery(nightly=True, **kwargs)
        report = self.run_discovery(nightly=True, replay=True)
        state = d.record_discovery(self.state, report)
        self.assertEqual(state['lanes'], {})
        self.assertTrue(any(c['channel'] == 'nightly' for c in report['candidates']))
    def test_fresh_aggregate_does_not_hide_stale_phase(self):
        self.source.update('docs/nightly/nightly.json', lambda x: x['provenance']['x86']['base'].update(builtAt='2026-08-13T22:00:00Z'))
        with self.assertRaisesRegex(d.DiscoveryError, 'stale nightly phase'): self.run_discovery(nightly=True)
        self.assertTrue(self.run_discovery(nightly=True, replay=True)['replay'])
    def test_missing_nightly_provenance_is_deferred(self):
        del self.source.blobs['docs/nightly/nightly.json']
        report = self.run_discovery(nightly=True)
        self.assertEqual(report['deferred'], [{'path': 'docs/nightly/nightly.json', 'reason': 'not-yet-published'}])
    def test_inspect_version_and_missing_metadata(self):
        self.source.update('docs/7.24.5/deep-inspect.json', lambda x: x['_meta'].update(version='7.24.4'))
        with self.assertRaises(d.DiscoveryError): self.run_discovery()
        self.source = Source()
        del self.source.blobs['docs/7.24.5/deep-inspect.json']
        with self.assertRaisesRegex(d.DiscoveryError, 'metadata missing'): self.run_discovery()
    def test_stage_checkpoint_recovery_no_false_generation(self):
        report = self.run_discovery(backfill=1)
        self.state = d.record_discovery(self.state, report)
        self.assertTrue(all(c['generation_needed'] for c in self.run_discovery(backfill=1)['candidates']))
        for c in report['candidates']:
            d.advance(self.state, c['key'], 'generated', d.encode({'candidate_key': c['key'], 'stage': 'generated', 'success': True}))
        self.assertEqual(self.run_discovery(backfill=1)['outcome'], 'unchanged')
        self.assertTrue(all(c['generation_needed'] for c in self.run_discovery(force=True)['candidates']))
        key = report['candidates'][0]['key']
        with self.assertRaises(d.DiscoveryError):
            d.advance(self.state, key, 'published', d.encode({'candidate_key': key, 'stage': 'published', 'success': True}))
        with self.assertRaises(d.DiscoveryError):
            d.advance(self.state, key, 'tested', d.encode({'candidate_key': key, 'stage': 'tested', 'success': False}))
        self.assertNotIn('tested', self.state['candidates'][key]['stages'])
        for stage in ('tested', 'published'):
            proof = d.encode({'candidate_key': key, 'stage': stage, 'success': True})
            d.advance(self.state, key, stage, proof)
            d.advance(self.state, key, stage, proof)
        with self.assertRaises(d.DiscoveryError):
            d.advance(self.state, key, 'published', d.encode({'candidate_key': key, 'stage': 'published', 'success': True, 'different': 1}))
    def test_dedup_excludes_commit_and_volatile_metadata_but_includes_semantics(self):
        report = self.run_discovery()
        first = report['candidates'][0]['key']
        self.source.update('docs/docs-index.json', lambda x: x.update(generatedAt='2026-10-02T12:00:00Z'))
        self.source.update('docs/7.25beta5/deep-inspect.json', lambda x: x['_meta'].update(generatedAt='2026-10-02T12:00:00Z'))
        self.assertEqual(first, self.run_discovery(requested='b' * 40)['candidates'][0]['key'])
        self.source.update('docs/7.25beta5/deep-inspect.json', lambda x: x.update(new_property={'_type': 'arg'}))
        self.assertNotEqual(first, self.run_discovery()['candidates'][0]['key'])
        report2, _ = d.discover(self.source, {'policy': 'v3'}, self.state, now=NOW)
        self.assertNotEqual(report2['candidates'][0]['key'], self.run_discovery()['candidates'][0]['key'])
    def test_regression_and_replay_do_not_downgrade_frontier(self):
        report = self.run_discovery()
        self.state = d.record_discovery(self.state, report)
        self.state['lanes']['stable/base/x86']['version'] = '7.24.6'
        with self.assertRaisesRegex(d.DiscoveryError, 'regressing'): self.run_discovery()
        replay = self.run_discovery(replay=True)
        self.assertEqual(d.record_discovery(self.state, replay)['lanes'], self.state['lanes'])
    def test_not_published_is_distinct_from_unchanged(self):
        report = self.run_discovery()
        self.state = d.record_discovery(self.state, report)
        for c in report['candidates']:
            d.advance(self.state, c['key'], 'generated', d.encode({'candidate_key': c['key'], 'stage': 'generated', 'success': True}))
        del self.source.blobs['docs/7.24.5/extra/openapi.json']
        # Other extra is also absent, so only already generated base candidates remain.
        del self.source.blobs['docs/7.25beta5/extra/openapi.json']
        self.assertEqual(self.run_discovery(extras=True)['outcome'], 'not-yet-published')
    def test_atomic_state_replacement(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'state.json'
            with d.locked(path):
                d.atomic(path, d.encode(self.state))
            self.assertEqual(d.load_state(path), self.state)
            self.assertFalse(path.with_name('state.json.tmp').exists())
    def test_committed_git_replay_and_failure_keeps_last_checkpoint(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            mirror = root / 'mirror'
            shutil.copytree(HERE / 'fixtures/docs', mirror / 'docs')
            def git(*args):
                return subprocess.run(['git', '-C', str(mirror), *args], check=True, capture_output=True).stdout
            git('init', '-q')
            git('add', 'docs')
            git('-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'test: snapshot')
            sha = git('rev-parse', 'HEAD').decode().strip()
            (mirror / 'docs/7.24.5/openapi.json').write_text('{}')
            source = d.GitSource(str(mirror))
            report, _ = d.discover(source, INPUTS, self.state, now=NOW)
            self.assertEqual(report['upstream_sha'], sha)
            state, output = root / 'state.json', root / 'output'
            command = [sys.executable, str(HERE / 'discover.py'), 'run', '--repository-dir', str(mirror), '--state', str(state), '--output', str(output)]
            self.assertEqual(subprocess.run(command, capture_output=True).returncode, 0)
            # Concurrent successful checkpoint writers must not lose either receipt.
            processes = []
            for key in d.load_state(state)['candidates']:
                evidence = root / (key + '.json')
                evidence.write_bytes(d.encode({'candidate_key': key, 'stage': 'generated', 'success': True}))
                processes.append(subprocess.Popen([sys.executable, str(HERE / 'discover.py'), 'checkpoint', '--state', str(state), '--key', key, '--stage', 'generated', '--evidence', str(evidence)], stdout=subprocess.PIPE, stderr=subprocess.PIPE))
            for process in processes:
                process.communicate(timeout=30)
                self.assertEqual(process.returncode, 0)
            self.assertTrue(all('generated' in c['stages'] for c in d.load_state(state)['candidates'].values()))
            before = state.read_bytes(), (output / 'manifest.json').read_bytes()
            git('add', 'docs')
            git('-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'test: malformed publication')
            failure = subprocess.run(command, capture_output=True)
            self.assertEqual(failure.returncode, 1)
            self.assertIn(b'discovery-failure:', failure.stderr)
            self.assertEqual(before, (state.read_bytes(), (output / 'manifest.json').read_bytes()))
    def test_http_retries_and_404_distinction(self):
        source = d.HTTPSource(attempts=3)
        error = urllib.error.HTTPError('https://example.invalid', 503, 'unavailable', {}, None)
        with patch.object(source.opener, 'open', side_effect=error) as fetch, patch.object(d.time, 'sleep'):
            with self.assertRaisesRegex(d.DiscoveryError, 'bounded retries'): source.fetch('https://example.invalid')
            self.assertEqual(fetch.call_count, 3)
        with patch.object(source.opener, 'open', side_effect=d.http.client.IncompleteRead(b'x', 20)) as fetch, patch.object(d.time, 'sleep'):
            with self.assertRaises(d.DiscoveryError): source.fetch('https://example.invalid')
            self.assertEqual(fetch.call_count, 3)
        missing = urllib.error.HTTPError('https://example.invalid', 404, 'missing', {}, None)
        with patch.object(source.opener, 'open', side_effect=missing) as fetch:
            with self.assertRaises(d.Missing): source.fetch('https://example.invalid')
            self.assertEqual(fetch.call_count, 1)
        for invalid in ['', 'main', 'abcd', 'A' * 40, 'a' * 40 + ';echo x']:
            with self.assertRaises(d.DiscoveryError): source.resolve(invalid)

if __name__ == '__main__':
    unittest.main()
