#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Extract offline path fixtures from committed restraml blobs, never its scripts."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))
import normalize as n

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--repository-dir', required=True)
parser.add_argument('--sha', default='adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a')
args = parser.parse_args()
source = n.discover.GitSource(args.repository_dir)
commit = source.resolve(args.sha)
records = {}
for version in ('7.24.5', '7.25beta5', '7.25_ab434'):
    for flavor in ('base', 'extra'):
        path = 'docs/' + ('nightly' if '_ab' in version else version) + ('/extra' if flavor == 'extra' else '') + '/openapi.json'
        raw = source.get(commit, path)
        spec = n.discover.parse(raw)
        subset = {k: spec[k] for k in ('openapi', 'info', 'components')}
        subset['paths'] = {p: spec['paths'][p] for p in ('/ip/address', '/ip/address/{id}')}
        data = (json.dumps(subset, sort_keys=True, indent=2) + '\n').encode()
        name = 'ip-address.upstream.json' if (version, flavor) == ('7.24.5', 'base') else version + '-' + flavor + '.upstream.json'
        (HERE / name).write_bytes(data)
        record = {'kind': 'extracted-offline-fixture', 'repository': n.discover.REPOSITORY, 'upstream_sha': commit,
                  'schema_path': path, 'upstream_full_sha256': hashlib.sha256(raw).hexdigest(),
                  'fixture_sha256': hashlib.sha256(data).hexdigest(),
                  'extraction': 'Unmodified OpenAPI info/components and two IP-address path objects, JSON reserialized. Not the complete upstream artifact.'}
        if name == 'ip-address.upstream.json':
            (HERE / 'provenance.json').write_text(json.dumps(record, sort_keys=True, indent=2) + '\n')
        records[name] = {**record, 'version': version, 'flavor': flavor}
(HERE / 'snapshot-fixtures.json').write_text(json.dumps(records, sort_keys=True, indent=2) + '\n')
