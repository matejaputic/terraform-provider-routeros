#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Extract reviewed networking paths from a pinned committed upstream blob."""
import argparse, hashlib, json, sys
from pathlib import Path
HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))
import normalize as n
p=argparse.ArgumentParser(description=__doc__)
p.add_argument('--repository-dir', required=True)
p.add_argument('--sha', default='adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a')
a=p.parse_args()
s=n.discover.GitSource(a.repository_dir);commit=s.resolve(a.sha)
raw=s.get(commit,'docs/7.24.5/openapi.json');spec=n.discover.parse(raw)
policies=n.discover.parse((n.ROOT/'internal/catalog/collections.json').read_bytes())
paths=['/ip/address']+[x['wire_path'] for x in policies]
subset={k:spec[k] for k in ('openapi','info','components')}
subset['paths']={route:spec['paths'][route] for path in paths for route in (path,path+'/{id}')}
data=n.discover.encode(subset)
(HERE/'collections.upstream.json').write_bytes(data)
(HERE/'collections.provenance.json').write_bytes(n.discover.encode({'kind':'extracted-offline-fixture','repository':n.discover.REPOSITORY,'upstream_sha':commit,'schema_path':'docs/7.24.5/openapi.json','upstream_full_sha256':hashlib.sha256(raw).hexdigest(),'fixture_sha256':n.sha(data),'extraction':'Unmodified info/components and reviewed networking collection/item paths, JSON reserialized; not the full upstream artifact.'}))
