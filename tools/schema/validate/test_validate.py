import copy
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]

class SpecificationGateTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.directory = tempfile.TemporaryDirectory()
        cls.binary = Path(cls.directory.name) / 'validate'
        subprocess.run(['go', 'build', '-o', str(cls.binary), 'tools/schema/validate/main.go'], cwd=ROOT,
                       env={**os.environ, 'GOTOOLCHAIN': 'go1.25.8'}, check=True, capture_output=True)
    @classmethod
    def tearDownClass(cls): cls.directory.cleanup()
    def setUp(self):
        self.spec = json.loads((ROOT / 'schemas/provider-code-spec.json').read_bytes())
        self.spec['resources'].sort(key=lambda r: r['name'] != 'ip_address')
    def check_spec(self, spec, valid=False):
        path = Path(self.directory.name) / 'input.json'
        path.write_text(json.dumps(spec))
        result = subprocess.run([str(self.binary), str(path)], capture_output=True)
        self.assertEqual(result.returncode == 0, valid, result.stderr.decode())
    def test_reviewed_spec_passes(self): self.check_spec(self.spec, valid=True)
    def test_zero_resources_extra_datasource_and_identity(self):
        original = copy.deepcopy(self.spec)
        self.spec['resources'] = []
        self.check_spec(self.spec)
        self.spec = copy.deepcopy(original)
        self.spec['data_sources'] = [{'name': 'ip_address'}]
        self.check_spec(self.spec)
        self.spec = copy.deepcopy(original)
        self.spec['provider']['name'] = 'scaffolding'
        self.check_spec(self.spec)
    def test_silent_omission_duplicates_wrong_type_flags(self):
        original = copy.deepcopy(self.spec)
        attrs = self.spec['resources'][0]['schema']['attributes']
        attrs.pop()
        self.check_spec(self.spec)
        self.spec = copy.deepcopy(original)
        attrs = self.spec['resources'][0]['schema']['attributes']
        attrs.append(copy.deepcopy(attrs[0]))
        self.check_spec(self.spec)
        self.spec = copy.deepcopy(original)
        attrs = self.spec['resources'][0]['schema']['attributes']
        disabled = next(a for a in attrs if a['name'] == 'disabled')
        disabled['string'] = disabled.pop('bool')
        self.check_spec(self.spec)
        self.spec = copy.deepcopy(original)
        self.spec['resources'][0]['schema']['attributes'][0]['string']['computed_optional_required'] = 'computed'
        self.check_spec(self.spec)
    def test_unions_instance_enums_and_secret_flags_rejected(self):
        original = copy.deepcopy(self.spec)
        self.spec['resources'][0]['schema']['attributes'][0]['bool'] = {'computed_optional_required': 'required'}
        self.check_spec(self.spec)
        for key, value in [('validators', [{'string': {'one_of': ['ether1', 'lo']}}]), ('sensitive', True), ('default', 'literal')]:
            self.spec = copy.deepcopy(original)
            self.spec['resources'][0]['schema']['attributes'][0]['string'][key] = value
            self.check_spec(self.spec)
    def test_partial_provider_or_extra_section_rejected(self):
        self.spec['provider']['schema'] = {'attributes': []}
        self.check_spec(self.spec)
        self.setUp()
        self.spec['unsupported'] = {}
        self.check_spec(self.spec)
    def test_duplicate_json_keys_are_not_silently_accepted(self):
        path = Path(self.directory.name) / 'input.json'
        raw = json.dumps(self.spec)
        path.write_text(raw[:-1] + ',"version":"0.1"}')
        self.assertNotEqual(subprocess.run([str(self.binary), str(path)], capture_output=True).returncode, 0)

if __name__ == '__main__': unittest.main()
