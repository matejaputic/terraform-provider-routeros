# SPDX-License-Identifier: MPL-2.0
"""Frozen source-identity lookup, never an exposure or field selector.

Exposure comes only from explicit maintained policy definitions. The frozen
ledger supplies constructor IDs for intentional canonical names (including names
which differ from reference aliases); candidates and statuses are not consumed.
"""
import json
from pathlib import Path
_ROOT=Path(__file__).resolve().parents[3]
_rows=json.loads((_ROOT/'schemas/resource-plan.json').read_text())['resources']
assert len(_rows)==len({r['constructor'] for r in _rows})==107
_IDENTITIES={r['canonical_name_proposal'].removeprefix('routeros_'):r['constructor'] for r in _rows}
def constructor(name):return _IDENTITIES[name]
