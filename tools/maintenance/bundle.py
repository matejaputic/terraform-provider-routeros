#!/usr/bin/env python3
"""Verified offline recovery bundles. Storage/authentication is a separate trust boundary."""
import argparse
import hashlib
import io
import os
from pathlib import Path, PurePosixPath
import re
import tarfile
import tempfile

import maintain as m

FORMAT = 'routeros-maintenance-bundle@1'
LIMIT = 2 * 1024 ** 3
MAX_FILES = 20000


def verify(directory):
    state = m.discovery.load_state(directory / 'checkpoints.json')
    observed = {}
    for path in (directory / 'runs').glob('*/discovery/manifest.json'):
        report = m.discovery.parse(path.read_bytes())
        sha = report.get('upstream_sha', '')
        m.require(re.fullmatch(r'[0-9a-f]{40}', sha) is not None, 'invalid snapshot revision')
        snapshot = path.parent / sha
        m.require(m.discovery.digest((snapshot / 'docs/docs-index.json').read_bytes())
                  == report['index_sha256'], 'snapshot index hash mismatch')
        for candidate in report['candidates']:
            identity = {k: candidate[k] for k in ('repository', 'channel', 'version', 'flavor',
                        'architecture', 'schema_path', 'schema_sha256', 'semantic_metadata', 'inputs')}
            m.require(m.discovery.digest(m.discovery.encode(identity)) == candidate['key'],
                      'snapshot candidate identity mismatch')
            artifacts = {candidate['schema_path']: candidate['schema_sha256']}
            artifacts.update({p: value['sha256'] for p, value in candidate['metadata'].items()})
            for name, expected in artifacts.items():
                m.discovery.valid_path(name)
                m.require(m.discovery.digest((snapshot / name).read_bytes()) == expected,
                          'snapshot artifact hash mismatch')
            observed[(candidate['key'], sha)] = candidate
    for key, entry in state['candidates'].items():
        sha = entry['stages'].get('discovered', {}).get('upstream_sha')
        m.require((key, sha) in observed, 'discovery receipt has no verified snapshot')
        candidate = observed[(key, sha)]
        m.require(entry.get('identity') == {k: candidate[k] for k in
                  ('version', 'channel', 'flavor', 'architecture', 'schema_sha256')},
                  'checkpoint identity differs from snapshot')
        m.require(not (set(entry['stages']) - {'discovered', 'generated'}),
                  'offline bundles cannot restore tested/published receipts')
        if 'generated' in entry['stages']:
            proof = m.discovery.parse((directory / 'candidates' / key / 'generated-evidence.json').read_bytes())
            m.require(re.fullmatch(r'[0-9a-f]{40}', proof.get('provider_revision', '')),
                      'invalid producer revision')
            m.require(re.fullmatch(r'[0-9a-f]{64}', proof.get('verification_inputs_sha256', '')),
                      'invalid verification binding')
            m.reusable(directory, {'key': key}, state, proof['verification_inputs_sha256'])
    return state


def selected(directory):
    paths = [directory / 'checkpoints.json']
    state = verify(directory)
    for key, entry in state['candidates'].items():
        if 'generated' in entry['stages']:
            paths.extend(p for p in (directory / 'candidates' / key).rglob('*') if p.is_file())
    # Preserve exact upstream manifests/raw bytes independently of ephemeral run logs.
    paths.extend(p for p in (directory / 'runs').glob('*/discovery/**/*') if p.is_file())
    m.require(all(not p.is_symlink() and not any(parent.is_symlink() for parent in p.parents)
                  for p in paths), 'bundle inputs must not contain symlinks')
    m.require(len(paths) <= MAX_FILES and sum(p.stat().st_size for p in paths) <= LIMIT,
              'bundle size limit exceeded')
    return sorted(paths)


def export(directory, archive):
    directory, archive = directory.resolve(), archive.resolve()
    m.require(not archive.is_relative_to(directory), 'archive must be outside maintenance output')
    archive.parent.mkdir(parents=True, exist_ok=True)
    with m.lock(directory):
        paths = selected(directory)
        files = {str(p.relative_to(directory)): m.discovery.digest(p.read_bytes()) for p in paths}
        manifest = m.discovery.encode({'format': FORMAT, 'artifacts': files})
        with tempfile.NamedTemporaryFile(dir=archive.parent, delete=False) as temporary:
            name = Path(temporary.name)
        try:
            with tarfile.open(name, 'w') as out:
                info = tarfile.TarInfo('bundle.json')
                info.size = len(manifest)
                out.addfile(info, io.BytesIO(manifest))
                for p in paths:
                    data = p.read_bytes()
                    m.require(m.discovery.digest(data) == files[str(p.relative_to(directory))],
                              'bundle input changed during export')
                    info = tarfile.TarInfo(str(p.relative_to(directory)))
                    info.size, info.mode = len(data), 0o600
                    out.addfile(info, io.BytesIO(data))
            # Immutable writes: an existing name is never replaced.
            os.link(name, archive)
        finally:
            name.unlink(missing_ok=True)
    return digest(archive)


def digest(path):
    with path.open('rb') as handle:
        return hashlib.file_digest(handle, 'sha256').hexdigest()


def restore(archive, directory, expected):
    m.require(re.fullmatch(r'[0-9a-f]{64}', expected) is not None, 'trusted archive digest required')
    m.require(archive.stat().st_size <= LIMIT + MAX_FILES * 1024, 'archive size limit exceeded')
    m.require(digest(archive) == expected, 'archive digest mismatch')
    directory = directory.resolve()
    directory.parent.mkdir(parents=True, exist_ok=True)
    m.require(not directory.exists(), 'restore requires a new output directory')
    with tempfile.TemporaryDirectory(dir=directory.parent, prefix='.restore-') as temporary:
        work = Path(temporary) / 'output'
        work.mkdir()
        names, total = set(), 0
        with tarfile.open(archive, 'r:') as source:
            for member in source:
                path = PurePosixPath(member.name)
                m.require(member.isfile() and not path.is_absolute() and '..' not in path.parts
                          and str(path) == member.name and member.name not in names,
                          'unsafe or duplicate archive member')
                names.add(member.name)
                total += member.size
                m.require(len(names) <= MAX_FILES + 1 and 0 <= member.size <= LIMIT and total <= LIMIT,
                          'expanded bundle size limit exceeded')
                target = work / member.name
                target.parent.mkdir(parents=True, exist_ok=True)
                with source.extractfile(member) as raw, target.open('xb') as out:
                    import shutil
                    shutil.copyfileobj(raw, out)
                target.chmod(0o600)
        manifest = m.discovery.parse((work / 'bundle.json').read_bytes())
        m.require(manifest.get('format') == FORMAT, 'unknown bundle format')
        (work / 'bundle.json').unlink()
        m.require(m.inventory(work) == manifest.get('artifacts'), 'bundle inventory mismatch')
        verify(work)
        m.require(set(manifest['artifacts']) == {str(p.relative_to(work)) for p in selected(work)},
                  'unexpected bundle artifacts')
        # Single rename exposes checkpoints and evidence together; failed validation changes nothing.
        m.require(not directory.exists(), 'restore destination appeared during verification')
        work.rename(directory)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['export', 'restore'])
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--archive', type=Path, required=True)
    parser.add_argument('--sha256')
    args = parser.parse_args()
    if args.command == 'export':
        print(export(args.output, args.archive))
    else:
        restore(args.archive, args.output, args.sha256 or '')


if __name__ == '__main__':
    main()
