# SPDX-License-Identifier: MPL-2.0
import json
from pathlib import Path
import unittest

import batch_a as b

class BatchAPlanningTest(unittest.TestCase):
 def setUp(self):
  self.matrix = b.BASELINE_MATRIX.read_bytes()
  self.contracts = (b.ROOT / 'schemas/reference-contracts/reference-contracts.json').read_bytes()

 def test_frozen_partition_is_exact_and_never_authorizes_exposure(self):
  ledger = b.build(self.matrix, self.contracts, {})
  matrix = json.loads(self.matrix)
  remaining = {r['constructor'] for r in matrix['constructors'] if not r['implemented']}
  a = {r['constructor'] for r in ledger['resources']}
  complement = set(ledger['batch_b_constructor_ids'])
  self.assertEqual(len(a), 107)
  self.assertEqual(len(complement), 107)
  self.assertFalse(a & complement)
  self.assertEqual(a | complement, remaining)
  self.assertEqual(len({r['canonical_name_proposal'] for r in ledger['resources']}), 107)
  for row in ledger['resources']:
   self.assertFalse(row['automatic_exposure_authorized'])
   self.assertIsNone(row['reviewed_contract'])
   self.assertEqual(row['implementation'], 'pending')
   self.assertEqual(row['official_generation'], 'pending')
   self.assertEqual(row['terraform_mock'], 'pending')
   self.assertEqual(row['live_evidence'], {})

 def test_immutable_baseline_tampering_is_rejected(self):
  with self.assertRaisesRegex(ValueError, 'hash mismatch'):
   b.build(self.matrix + b'\n', self.contracts, {})

 def test_wrong_static_reference_revision_is_rejected(self):
  with self.assertRaisesRegex(ValueError, 'static reference revision mismatch'):
   b.build(self.matrix, b'{"source":{"revision":"unreviewed"}}', {})

 def test_incomplete_and_missing_paths_are_not_invented(self):
  paths = {'/user/ssh-keys': {'get': {}, 'put': {}}, '/user/ssh-keys/{id}': {'get': {}, 'delete': {}}}
  partial = b.schema_shape({'paths': paths}, '/user/ssh-keys')
  self.assertFalse(partial['structural_collection_crud'])
  self.assertTrue(partial['path_present'])
  self.assertNotIn('patch', partial['item_methods'])
  missing = b.schema_shape({'paths': paths}, '/interface/veth')
  self.assertFalse(missing['path_present'])
  self.assertFalse(missing['structural_collection_crud'])

 def test_persisted_ledger_has_correct_bindings_and_four_observation_lanes(self):
  ledger = json.loads((b.ROOT / 'schemas/batch-a-ledger.json').read_text())
  self.assertEqual(ledger['baseline_matrix_sha256'], b.MATRIX_SHA256)
  self.assertEqual(ledger['baseline_source'], b.BASELINE)
  self.assertEqual(ledger['upstream_sha'], b.UPSTREAM)
  lanes = {'7.24.5/base', '7.24.5/extra', '7.25beta5/base', '7.25beta5/extra'}
  self.assertEqual(set(ledger['schema_inputs']), lanes)
  self.assertEqual(len(ledger['resources']), 107)
  rows = {r['constructor']: r for r in ledger['resources']}
  for row in rows.values():
   self.assertEqual(set(row['schema_observations']), lanes)
   self.assertFalse(row['automatic_exposure_authorized'])
  for version in ('7.24.5', '7.25beta5'):
   self.assertFalse(rows['ResourceInterfaceVeth']['schema_observations'][version + '/base']['path_present'])
   self.assertTrue(rows['ResourceInterfaceVeth']['schema_observations'][version + '/extra']['structural_collection_crud'])
   for flavor in ('base', 'extra'):
    self.assertNotIn('patch', rows['ResourceUserSshKeys']['schema_observations'][version + '/' + flavor]['item_methods'])
  self.assertEqual(rows['ResourceDhcpServerOptionSets']['canonical_name_proposal'], 'routeros_ip_dhcp_server_option_set')

if __name__ == '__main__':
 unittest.main()
