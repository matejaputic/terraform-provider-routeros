"""Regenerate small synthetic restraml-docs-index@1 fixtures (not real schemas)."""
import json
from pathlib import Path

ROOT = Path(__file__).parent
VERSIONS = ['7.24.5', '7.25beta5', '7.25rc1', '7.24.4', 'nightly']
TIME = '2026-10-01T23:00:00Z'

def save(path, value):
    target = ROOT / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(value, sort_keys=True, indent=2) + '\n')

def file(directory, name):
    return {'name': name, 'path': directory + '/' + name, 'type': 'file'}

entries = []
for name in VERSIONS:
    directory = 'docs/' + name
    version = '7.25_ab434' if name == 'nightly' else name
    for flavor in ('base', 'extra'):
        path = directory + ('/extra' if flavor == 'extra' else '')
        save(path + '/openapi.json', {'openapi': '3.0.3', 'info': {'version': version, 'title': 'Synthetic fixture'}, 'paths': {'/ip/address': {'get': {'responses': {'200': {'description': 'Synthetic'}}}}}})
    save(directory + '/deep-inspect.json', {'_meta': {'version': version, 'generatedAt': TIME, 'architecture': 'x86'}, 'ip': {'_type': 'path'}})
    entries.append({'name': name, 'path': directory, 'type': 'dir',
                    'files': [file(directory, 'openapi.json'), file(directory, 'deep-inspect.json')],
                    'dirs': [{'name': 'extra', 'path': directory + '/extra', 'type': 'dir',
                              'files': [file(directory + '/extra', 'openapi.json')], 'dirs': []}]})
provenance = {}
for arch, phases in [('x86', ['base', 'extra']), ('arm64', ['extra'])]:
    provenance[arch] = {phase: {'arch': arch, 'phase': phase, 'nightlyVersion': '7.25_ab434',
                               'postVersion': '7.25_ab434', 'builtAt': '2026-10-01T22:00:00Z'} for phase in phases}
save('docs/nightly/nightly.json', {'nightlyVersion': '7.25_ab434', 'builtAt': TIME, 'provenance': provenance})
save('docs/docs-index.json', {'format': 'restraml-docs-index@1', 'generatedAt': '2026-10-02T00:00:00Z', 'rootPath': 'docs',
                              'latestStableVersion': '7.24.5', 'latestVersion': '7.25beta5',
                              'nightly': {'nightlyVersion': '7.25_ab434', 'builtAt': TIME}, 'versions': entries, 'files': []})
