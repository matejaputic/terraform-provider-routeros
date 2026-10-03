import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import normalize as n
import verify as bundle
from datetime import datetime, timezone
import test_discover

ROOT = n.ROOT
FIXTURE = Path(__file__).parent / 'fixtures/ip-address.upstream.json'
POLICY = ROOT / 'schemas/ip-address-policy.json'

class NormalizationTests(unittest.TestCase):
    def setUp(self):
        self.spec = json.loads(FIXTURE.read_bytes())
        self.policy = json.loads(POLICY.read_bytes())
    def adapt(self, **kwargs):
        return n.normalize(n.discover.encode(self.spec), self.policy, **kwargs)
    def create(self):
        return self.spec['paths']['/ip/address']['put']['requestBody']['content']['application/json']['schema']['allOf'][0]
    def update(self):
        return self.spec['paths']['/ip/address/{id}']['patch']['requestBody']['content']['application/json']['schema']['allOf'][0]
    def read(self):
        return self.spec['paths']['/ip/address/{id}']['get']['responses']['200']['content']['application/json']
    def test_baseline_golden_outputs_and_types(self):
        spec, config, wire, report = n.normalize(FIXTURE.read_bytes(), self.policy,
            origin=json.loads((FIXTURE.parent / 'provenance.json').read_bytes()))
        combined = json.loads((ROOT / 'schemas/ip-address.openapi.json').read_bytes())
        self.assertEqual(spec['paths'], {p: combined['paths'][p] for p in spec['paths']})
        self.assertTrue((ROOT / 'schemas/generator-config.yml').read_bytes().startswith(config))
        self.assertEqual(wire['resources'][0], json.loads((ROOT / 'schemas/wire-descriptors.json').read_bytes())['resources'][0])
        self.assertEqual(report['adaptations'], json.loads((ROOT / 'schemas/adaptation-report.json').read_bytes())['adaptations'])
        self.assertEqual(self.adapt()[0], self.adapt()[0])
        body = spec['paths']['/ip/address']['put']['requestBody']['content']['application/json']['schema']
        self.assertEqual(body['required'], ['address', 'interface'])
        self.assertEqual(body['properties']['disabled'], {'type': 'boolean'})
        self.assertNotIn('enum', body['properties']['interface'])
        self.assertNotIn('vrf', body['properties'])
        self.assertNotIn('parameters', spec['paths']['/ip/address/{id}']['get'])
        self.assertEqual(wire['resources'][0]['runtime_read']['id_filter'], '.id')
    def test_pinned_stable_extra_beta_nightly_snapshot_fixtures(self):
        records = json.loads((FIXTURE.parent / 'snapshot-fixtures.json').read_bytes())
        self.assertEqual(len(records), 6)
        baseline = self.adapt()[0]
        for name, record in records.items():
            with self.subTest(version=record['version'], flavor=record['flavor']):
                raw = (FIXTURE.parent / name).read_bytes()
                self.assertEqual(n.sha(raw), record['fixture_sha256'])
                spec, _, _, report = n.normalize(raw, self.policy, origin=record)
                expected = copy.deepcopy(baseline)
                expected['info']['version'] = record['version']
                self.assertEqual(spec, expected)
                self.assertEqual(report['supported_resources'], ['ip_address'])
                self.assertEqual(report['data_sources'], [])
    def test_internal_refs_nested_allof_and_required_union(self):
        resolver = n.Resolver({'components': {'schemas': {'A': {'type': 'object', 'properties': {'x': {'type': 'string'}}, 'required': ['x']}}}})
        result = resolver.resolve({'allOf': [{'$ref': '#/components/schemas/A'}, {'allOf': [{'type': 'object', 'properties': {'y': {'type': 'boolean'}}, 'required': ['y']}]}]})
        self.assertEqual(result['required'], ['x', 'y'])
        self.assertEqual(set(result['properties']), {'x', 'y'})
        self.spec['components']['schemas']['Create'] = copy.deepcopy(self.create())
        self.spec['paths']['/ip/address']['put']['requestBody']['content']['application/json']['schema'] = {'allOf': [{'$ref': '#/components/schemas/Create'}, {'$ref': '#/components/schemas/QueryOptions'}]}
        self.adapt()
        body = self.spec['paths']['/ip/address']['put']['requestBody']
        self.spec['components']['requestBodies'] = {'A': body, 'B': {'$ref': '#/components/requestBodies/A'}}
        self.spec['paths']['/ip/address']['put']['requestBody'] = {'$ref': '#/components/requestBodies/B'}
        response = self.spec['paths']['/ip/address/{id}']['get']['responses']['200']
        self.spec['components']['responses']['ReadA'] = response
        self.spec['components']['responses']['ReadB'] = {'$ref': '#/components/responses/ReadA'}
        self.spec['paths']['/ip/address/{id}']['get']['responses']['200'] = {'$ref': '#/components/responses/ReadB'}
        self.adapt()
    def test_conflict_external_recursive_refs_and_unsupported_union(self):
        bad = [
            {'allOf': [{'type': 'object', 'properties': {'x': {'type': 'string'}}}, {'type': 'object', 'properties': {'x': {'type': 'boolean'}}}]},
            {'$ref': 'https://untrusted.invalid/schema'}, {'$ref': '#/missing'},
            {'anyOf': [{'type': 'string'}, {'type': 'array'}]},
            {'allOf': [{'type': 'array', 'items': {'type': 'string'}}]}, {'type': ['string', 'null']},
        ]
        for schema in bad:
            with self.subTest(schema=schema), self.assertRaises(n.AdaptationError): n.Resolver(self.spec).resolve(schema)
        self.spec['components']['schemas']['Cycle'] = {'$ref': '#/components/schemas/Cycle'}
        with self.assertRaises(n.AdaptationError): n.Resolver(self.spec).resolve({'$ref': '#/components/schemas/Cycle'})
    def test_query_controls_removed_before_union_resolution(self):
        self.create()['properties']['.proplist'] = {'oneOf': [{'type': 'string'}, {'type': 'array'}]}
        self.create()['required'] = ['.proplist']
        spec, _, _, report = self.adapt()
        self.assertIn('shared QueryOptions', report['adaptations']['removed_controls'])
        self.assertIn('.proplist', report['adaptations']['removed_controls'])
        self.assertNotIn('.proplist', json.dumps(spec))
        del self.spec['components']['schemas']['QueryOptions']
        with self.assertRaisesRegex(n.AdaptationError, 'unresolved internal ref'): self.adapt()
    def test_unknown_field_is_inventory_only_and_static_enum_unreviewed(self):
        self.create()['properties']['future-field'] = {'type': 'string', 'enum': ['one', 'two']}
        spec, _, _, report = self.adapt()
        self.assertNotIn('future_field', json.dumps(spec))
        self.assertIn('future-field', report['adaptations']['unreviewed_wire_string_candidates'])
    def test_absence_and_empty_read_response_retain_catalog(self):
        self.read()['schema'] = {'type': 'object', 'properties': {}}
        del self.create()['properties']['network']
        del self.update()['properties']['network']
        spec, _, _, report = self.adapt()
        self.assertIn('network', report['adaptations']['retained_curated_fields_absent_from_observation'])
        props = spec['paths']['/ip/address/{id}']['get']['responses']['200']['content']['application/json']['schema']['properties']
        self.assertEqual(len(props), 11)
        self.assertEqual(props['dynamic'], {'type': 'boolean'})
    def test_create_only_known_field_and_lost_crud_fail(self):
        del self.update()['properties']['address']
        with self.assertRaisesRegex(n.AdaptationError, 'create-only'): self.adapt()
        self.setUp()
        del self.spec['paths']['/ip/address/{id}']['delete']
        with self.assertRaisesRegex(n.AdaptationError, 'CRUD'): self.adapt()
    def test_reviewed_type_and_boolean_domain_drift_fail(self):
        self.create()['properties']['address']['type'] = 'integer'
        with self.assertRaisesRegex(n.AdaptationError, 'type drift'): self.adapt()
        self.setUp()
        self.update()['properties']['disabled']['enum'] = ['maybe']
        with self.assertRaisesRegex(n.AdaptationError, 'boolean wire domain'): self.adapt()
    def test_tf_go_id_and_resource_collisions(self):
        for names in [{'foo-bar', 'foo_bar'}, {'.id', 'id'}, {'foo_bar', 'foo__bar'}, {'a_bc', 'ab_c'}]:
            # Last pair is distinct in Terraform and Go; verify no over-rejection.
            if names == {'a_bc', 'ab_c'}:
                self.assertEqual(len(n.symbols(names)), 2)
            else:
                with self.assertRaises(n.AdaptationError): n.symbols(names)
        self.create()['properties']['id'] = {'type': 'string'}
        with self.assertRaisesRegex(n.AdaptationError, 'collision'): self.adapt()
        self.setUp()
        for p in ['/ip-address', '/ip_address']:
            self.spec['paths'][p] = copy.deepcopy(self.spec['paths']['/ip/address'])
            self.spec['paths'][p + '/{id}'] = copy.deepcopy(self.spec['paths']['/ip/address/{id}'])
        with self.assertRaisesRegex(n.AdaptationError, 'collision'): self.adapt()
    def test_actions_singletons_and_hardware_are_never_exposed(self):
        self.spec['paths']['/system/reboot'] = {'post': {'responses': {'200': {'description': 'Action'}}}}
        self.spec['paths']['/system/identity'] = {'get': {'responses': {'200': {'description': 'Settings'}}}, 'patch': {'responses': {'200': {'description': 'Settings'}}}}
        self.spec['paths']['/interface/ethernet'] = copy.deepcopy(self.spec['paths']['/ip/address'])
        self.spec['paths']['/interface/ethernet/{id}'] = copy.deepcopy(self.spec['paths']['/ip/address/{id}'])
        spec, _, _, report = self.adapt()
        self.assertEqual(set(spec['paths']), {'/ip/address', '/ip/address/{id}'})
        self.assertEqual(report['supported_resources'], ['ip_address'])
        classes = {e['classification'] for e in report['catalog']}
        self.assertTrue({'action-excluded', 'singleton-or-existing-object-deferred', 'existing-hardware-deferred'} <= classes)
    def test_metadata_suggestions_are_non_authoritative_and_version_bound(self):
        metadata = {'deep-inspect.json': n.discover.encode({'_meta': {'version': '7.24.5'}, 'properties': {'future-field': {'type': 'boolean'}}})}
        spec, _, _, report = self.adapt(metadata=metadata)
        self.assertNotIn('future_field', json.dumps(spec))
        self.assertIn('future-field', report['adaptations']['unreviewed_wire_string_candidates'])
        metadata['deep-inspect.json'] = n.discover.encode({'_meta': {'version': '7.25beta5'}})
        with self.assertRaises(n.AdaptationError): self.adapt(metadata=metadata)
    def test_unapproved_unsupported_body_is_reported_not_exposed(self):
        self.spec['paths']['/new'] = copy.deepcopy(self.spec['paths']['/ip/address'])
        self.spec['paths']['/new/{id}'] = copy.deepcopy(self.spec['paths']['/ip/address/{id}'])
        self.spec['paths']['/new']['put']['requestBody']['content']['application/json']['schema'] = {'oneOf': [{'type': 'object'}]}
        _, _, _, report = self.adapt()
        entry = next(e for e in report['catalog'] if e['path'] == '/new' and e['classification'] != 'action-excluded')
        self.assertEqual(entry['adaptation_status'], 'unsupported-needs-review')
        self.assertEqual(report['supported_resources'], ['ip_address'])
    def test_cli_preserves_raw_bytes_and_failure_preserves_outputs(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'output'
            source = Path(directory) / 'raw.json'
            source.write_bytes(FIXTURE.read_bytes())
            args = [sys.executable, str(Path(n.__file__)), '--input', str(source), '--output', str(output)]
            sentinel = 'not-a-real-provider-secret-step3'
            result = subprocess.run(args, capture_output=True, env={**os.environ, 'ROS_PASSWORD': sentinel, 'MIKROTIK_PASSWORD': sentinel})
            self.assertEqual(result.returncode, 0)
            self.assertNotIn(sentinel.encode(), result.stdout + result.stderr)
            for artifact in output.iterdir():
                self.assertNotIn(sentinel.encode(), artifact.read_bytes())
            self.assertEqual((output / 'upstream-input.json').read_bytes(), source.read_bytes())
            before = {p.name: p.read_bytes() for p in output.glob('*.json')}
            source.write_text('{}')
            result = subprocess.run(args, capture_output=True)
            self.assertEqual(result.returncode, 1)
            self.assertIn(b'adaptation-failure:', result.stderr)
            self.assertEqual(before, {p.name: p.read_bytes() for p in output.glob('*.json')})
    def test_bundle_hashes_reject_partial_outputs(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            spec, config, wire, report = self.adapt()
            for name, data in [('upstream-input.json', n.discover.encode(self.spec)), ('ip-address.openapi.json', n.discover.encode(spec)), ('generator-config.yml', config), ('wire-descriptors.json', n.discover.encode(wire)), ('adaptation-report.json', n.discover.encode(report))]:
                (output / name).write_bytes(data)
            bundle.verify(output)
            (output / 'generator-config.yml').write_text('provider: broken')
            with self.assertRaisesRegex(n.AdaptationError, 'partial/tampered'): bundle.verify(output)
    def test_discovery_contract_hash_and_metadata_integrity(self):
        source = test_discover.Source()
        for path in list(source.blobs):
            if path.endswith('/openapi.json'):
                obj = copy.deepcopy(self.spec)
                obj['info']['version'] = '7.25_ab434' if '/nightly/' in path else path.split('/')[1]
                source.blobs[path] = n.discover.encode(obj)
        report, blobs = n.discover.discover(source, n.discover.fingerprints(ROOT), {'format': n.discover.FORMAT, 'candidates': {}, 'lanes': {}}, now=datetime(2026, 10, 2, tzinfo=timezone.utc))
        candidate = next(c for c in report['candidates'] if c['version'] == '7.24.5')
        with tempfile.TemporaryDirectory() as directory:
            snapshot, output = Path(directory) / 'snapshot', Path(directory) / 'output'
            snapshot.mkdir()
            manifest = snapshot / 'manifest.json'
            manifest.write_bytes(n.discover.encode(report))
            for path, raw in blobs.items():
                target = snapshot / report['upstream_sha'] / path
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(raw)
            raw = snapshot / report['upstream_sha'] / candidate['schema_path']
            args = [sys.executable, str(Path(n.__file__)), '--manifest', str(manifest), '--key', candidate['key'], '--input', str(raw), '--output', str(output)]
            result = subprocess.run(args, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr.decode())
            self.assertEqual(bundle.verify(output)['origin']['key'], candidate['key'])
            before = (output / 'adaptation-report.json').read_bytes()
            raw.write_text('{}')
            self.assertNotEqual(subprocess.run(args, capture_output=True).returncode, 0)
            raw.write_bytes(blobs[candidate['schema_path']])
            metadata_path = next(iter(candidate['metadata']))
            (snapshot / report['upstream_sha'] / metadata_path).write_text('{}')
            self.assertNotEqual(subprocess.run(args, capture_output=True).returncode, 0)
            self.assertEqual(before, (output / 'adaptation-report.json').read_bytes())
    def test_schema_version_sensitive_and_enum_changes_need_explicit_runtime_work(self):
        for key, value in [('sensitive', True), ('force_new', True), ('enum_kind', 'static')]:
            self.setUp()
            self.policy['attributes'][0][key] = value
            with self.assertRaises(n.AdaptationError): self.adapt()
        self.setUp()
        self.policy['schema_version'] = 1
        with self.assertRaises(n.AdaptationError): self.adapt()

if __name__ == '__main__':
    unittest.main()
