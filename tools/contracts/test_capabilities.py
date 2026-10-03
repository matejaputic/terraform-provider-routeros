import copy
import json
from pathlib import Path
import tempfile
import unittest

import capabilities as c
import extract


class CapabilityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.before = c.load(c.ROOT / 'schemas/wire-descriptors.json')
        cls.policy = c.load(c.POLICY)

    def assess(self, after):
        return c.assess(self.before, after, self.policy)

    def test_identity_passes_offline_only(self):
        report = self.assess(copy.deepcopy(self.before))
        self.assertEqual(report['decision'], 'offline-compatible')
        for name in ('new_exposure_authorized', 'live_tested', 'publication_authorized'):
            self.assertFalse(report[name])

    def test_only_provenance_and_identity_order_are_ignored(self):
        after = copy.deepcopy(self.before)
        after['resources'].reverse()
        for r in after['resources']:
            r['fields'].reverse()
            for field in r['fields']:
                field['provenance'] = 'new immutable source evidence'
        self.assertEqual(self.assess(after)['decision'], 'offline-compatible')

    def test_field_semantics_all_require_review(self):
        for key, value in {'type': 'integer', 'mode': 'required', 'wire': 'changed', 'codec': 'unknown',
                           'default': 'invented', 'force_new': True, 'sensitive': True,
                           'validators': 'new validator', 'enum_kind': 'new enum',
                           'novel_property': 'unexpected'}.items():
            with self.subTest(key=key):
                after = copy.deepcopy(self.before)
                after['resources'][0]['fields'][0][key] = value
                report = self.assess(after)
                self.assertEqual(report['decision'], 'review-required')
                self.assertTrue(any(f['path'].endswith('/' + key) for f in report['findings']))

    def test_resource_semantics_all_require_review(self):
        for key, value in {'path': '/unknown', 'id_wire_key': 'name', 'schema_version': 1,
                           'lifecycle': 'singleton', 'migration_compatibility': 'claimed',
                           'operations': {}, 'runtime_read': {}, 'schema_read': {},
                           'new_semantics': True}.items():
            with self.subTest(key=key):
                after = copy.deepcopy(self.before)
                after['resources'][0][key] = value
                self.assertEqual(self.assess(after)['decision'], 'review-required')

    def test_additions_removals_and_empty_contracts_require_review(self):
        for action in ('resource-add', 'resource-remove', 'field-add', 'field-remove', 'empty'):
            after = copy.deepcopy(self.before)
            if action == 'resource-add':
                new = copy.deepcopy(after['resources'][0]); new['terraform_type'] = 'routeros_new'
                after['resources'].append(new)
            elif action == 'resource-remove':
                after['resources'].pop()
            elif action == 'field-add':
                new = copy.deepcopy(after['resources'][0]['fields'][0]); new['name'] = 'new'
                after['resources'][0]['fields'].append(new)
            elif action == 'field-remove':
                after['resources'][0]['fields'].pop()
            else:
                after['resources'] = []
            self.assertEqual(self.assess(after)['decision'], 'review-required')

    def test_duplicate_identities_and_unreviewed_baseline_fail(self):
        after = copy.deepcopy(self.before)
        after['resources'].append(after['resources'][0])
        with self.assertRaises(ValueError):
            self.assess(after)
        before = copy.deepcopy(self.before)
        before['resources'][0]['path'] = '/unknown'
        with self.assertRaisesRegex(ValueError, 'reviewed policy'):
            c.assess(before, before, self.policy)
        policy = dict(self.policy, revision='unreviewed')
        with self.assertRaisesRegex(ValueError, 'unreviewed'):
            c.assess(self.before, self.before, policy)

    def test_type_change_bool_not_equal_to_int(self):
        after = copy.deepcopy(self.before)
        after['resources'][0]['fields'][0]['force_new'] = 0
        self.assertEqual(self.assess(after)['decision'], 'review-required')

    def test_novel_envelope_keys_and_revision_changes_require_review(self):
        after = copy.deepcopy(self.before)
        after['novel'] = True
        self.assertEqual(self.assess(after)['decision'], 'review-required')
        after = copy.deepcopy(self.before)
        after['policy_revision'] = 'new'
        self.assertEqual(self.assess(after)['decision'], 'review-required')

    def test_json_duplicate_and_nonfinite_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            p = Path(directory) / 'bad.json'
            for text in ('{"same":1,"same":2}', '{"value":NaN}'):
                p.write_text(text)
                with self.assertRaises(ValueError):
                    c.load(p)

    def test_callback_classification_does_not_execute_or_approve(self):
        for prefix, expected in (('Default', 'ordinary-collection-candidate'),
                                 ('DefaultSystem', 'singleton-candidate'),
                                 ('updateOnlyDevice', 'hardware-update-only'),
                                 ('Custom', 'custom-or-unresolved')):
            contract = {'resource_properties': {name + 'Context': {'call': {'callee': prefix + name}}
                                               for name in ('Create', 'Read', 'Update', 'Delete')}}
            self.assertEqual(c.lifecycle(contract)[0], expected)
        self.assertEqual(c.lifecycle({'resource_properties': {}})[0], 'custom-or-unresolved')

    def test_full_matrix_accounts_for_aliases_and_existing_subsets(self):
        matrix = c.matrix(c.ROOT / 'schemas/reference-contracts', self.before)
        self.assertEqual(matrix['counts']['resource_names'], 256)
        self.assertEqual(matrix['counts']['resource_constructors'], 232)
        self.assertEqual(matrix['counts']['implemented_subsets'], len(self.before['resources']))
        exposed_names = {item['resource'] for row in matrix['constructors'] for item in row['implemented']}
        self.assertEqual(exposed_names, {item['terraform_type'] for item in self.before['resources']})
        for row in matrix['constructors']:
            self.assertTrue(row['semantic_review_required'])
            self.assertFalse(row['automatic_exposure_authorized'])
        self.assertEqual(extract.encode(matrix), extract.encode(c.matrix(c.ROOT / 'schemas/reference-contracts', self.before)))
        self.assertEqual(extract.encode(matrix), extract.encode(c.load(c.ROOT / 'schemas/capability-matrix.json')))
        exposed = [item for row in matrix['constructors'] for item in row['implemented']]
        self.assertTrue(all(field['maintained_codec_helper'] for item in exposed for field in item['field_capabilities']))

    def test_modified_bundle_and_stale_overlay_fail_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            import shutil
            for name in ('manifest.json', 'reference-contracts.json', 'reconciliation.json'):
                shutil.copyfile(c.ROOT / 'schemas/reference-contracts' / name, root / name)
            (root / 'reconciliation.json').write_text('{}')
            with self.assertRaises(ValueError):
                c.matrix(root, self.before)
        before = copy.deepcopy(self.before)
        before['resources'].pop()
        with self.assertRaisesRegex(ValueError, 'stale'):
            c.matrix(c.ROOT / 'schemas/reference-contracts', before)


if __name__ == '__main__':
    unittest.main()
