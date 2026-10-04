#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Freeze remaining constructor identities; never authorize public exposure."""
import hashlib,json,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
def freeze():
 matrix_path=ROOT/'schemas/capability-matrix.json';matrix=json.loads(matrix_path.read_text())
 historical=json.loads((ROOT/'schemas/resource-plan.json').read_text())
 remaining=[row for row in matrix['constructors'] if not row['implemented']]
 if len(remaining)!=107 or {r['constructor'] for r in remaining}!=set(historical['remaining_constructor_ids']):raise ValueError('Remaining constructors differ from the frozen resource partition')
 if matrix['counts']['implemented_subsets']!=125:raise ValueError('requires the completed 125-constructor baseline')
 rows=[{'constructor':r['constructor'],'reference_names':r['resource_names'],'reference_source':r['source'],'static_family':r['family'],'wire_path_observation':r['wire_path_candidate'],'risks':r['risks'],'automatic_exposure_authorized':False,'implementation_status':'pending explicit maintained contract and lifecycle review'} for r in remaining]
 return {'format':'routeros-remaining-resource-plan@1','baseline_source':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'baseline_implemented':125,'remaining_constructor_count':107,'target_total':232,'reference_sha':matrix['reference_sha'],'baseline_matrix_sha256':hashlib.sha256(matrix_path.read_bytes()).hexdigest(),'historical_partition_sha256':hashlib.sha256((ROOT/'schemas/resource-plan.json').read_bytes()).hexdigest(),'scope':'frozen constructor identities only; aliases are not separate integrations; no inventory/ledger row authorizes exposure','resources':rows}
if __name__=='__main__':
 target=ROOT/'schemas/remaining-resource-plan.json'
 if target.exists():raise SystemExit('refusing to replace the frozen remaining-resource plan')
 target.write_text(json.dumps(freeze(),indent=2,sort_keys=True)+'\n')
