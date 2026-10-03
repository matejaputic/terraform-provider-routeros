#!/usr/bin/env python3
"""Artifact-only immutable maintenance. Never advances live-tested/published receipts."""
import argparse
from contextlib import contextmanager
from datetime import datetime, timezone
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import time
import uuid

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('discovery', ROOT / 'tools/schema/discover/discover.py')
discovery = importlib.util.module_from_spec(spec)
spec.loader.exec_module(discovery)
FORMAT = 'routeros-maintenance@1'
ENV_PINS = {'GOTOOLCHAIN': 'go1.25.8', 'TF_ACC_TERRAFORM_VERSION': '1.14.0'}


class MaintenanceError(Exception):
    pass


def require(ok, message):
    if not ok:
        raise MaintenanceError(message)


def run(command, cwd, log, timeout=1200):
    # Do not inherit credentials into offline tests/builds. Live tests are never enabled.
    env = {k: v for k, v in os.environ.items()
           if not k.startswith(('ROS_', 'MIKROTIK_', 'TF_ACC'))
           and k not in ('GITHUB_TOKEN', 'GH_TOKEN', 'ACTIONS_ID_TOKEN_REQUEST_TOKEN')}
    env.update(ENV_PINS)
    with log.open('ab') as out:
        result = subprocess.run(command, cwd=cwd, env=env, stdout=out,
                                stderr=subprocess.STDOUT, timeout=timeout)
    require(result.returncode == 0, 'maintenance command failed; see offline run log')


def tracked_revision(root):
    status = subprocess.check_output(['git', 'status', '--porcelain'], cwd=root)
    require(not status, 'maintenance requires a clean committed provider checkout')
    return subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()


def verification_inputs(root):
    # Discovery binds runtime/adapter/generator inputs; additionally invalidate offline
    # verification reuse on tests, fixtures, build policy or orchestration changes.
    files = subprocess.check_output(['git', 'ls-files', '-z'], cwd=root).decode().split('\0')
    included = [p for p in files if p and not p.endswith('.md')
                and (p.startswith(('internal/', 'tools/', '.github/workflows/'))
                     or p in ('GNUmakefile', 'go.mod', 'go.sum',
                              'schemas/maintenance-policy.json', 'schemas/wire-descriptors.json'))]
    return discovery.digest(discovery.encode({p: discovery.digest((root / p).read_bytes())
                                             for p in sorted(included)}))


@contextmanager
def lock(output):
    output.mkdir(parents=True, exist_ok=True)
    with (output / '.maintenance.lock').open('a') as handle:
        try:
            fcntl.flock(handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise MaintenanceError('another maintenance run owns this output') from None
        yield


def inventory(directory):
    return {str(p.relative_to(directory)): discovery.digest(p.read_bytes())
            for p in sorted(directory.rglob('*')) if p.is_file()}


def reusable(output, candidate, state, verification):
    entry = state.get('candidates', {}).get(candidate['key'], {})
    receipt = entry.get('stages', {}).get('generated')
    if not receipt:
        return False
    directory = output / 'candidates' / candidate['key']
    proof_path = directory / 'generated-evidence.json'
    require(proof_path.is_file(), 'successful receipt has no local artifact evidence; restore its bundle')
    raw = proof_path.read_bytes()
    proof = discovery.parse(raw)
    require(discovery.digest(raw) == receipt['evidence_sha256'], 'generated evidence differs from checkpoint')
    require(proof.get('candidate_key') == candidate['key'] and proof.get('stage') == 'generated'
            and proof.get('success') is True, 'invalid generated evidence')
    require(proof.get('verification_inputs_sha256') == verification,
            'offline verification inputs changed; use a new output namespace')
    files = inventory(directory)
    files.pop('generated-evidence.json', None)
    require(files == proof.get('artifacts'), 'successful candidate artifacts are missing or modified')
    return True


def produce(root, archive, candidate, manifest, snapshot, destination, log):
    with tempfile.TemporaryDirectory(prefix='routeros-maintenance-') as name:
        work = Path(name)
        with tarfile.open(archive) as source:
            source.extractall(work, filter='data')
        baseline = work / '.approved-descriptors.json'
        baseline.write_bytes((work / 'schemas/wire-descriptors.json').read_bytes())
        run(['bash', 'tools/schema/generate.sh', '--manifest', str(manifest), '--key', candidate['key'],
             '--input', str(snapshot / candidate['upstream_sha'] / candidate['schema_path'])], work, log)
        first = {str(p.relative_to(work)): discovery.digest(p.read_bytes())
                 for folder in ('schemas', 'internal/generated')
                 for p in (work / folder).rglob('*') if p.is_file() and p.name != 'adaptation.lock'}
        run(['bash', 'tools/schema/generate.sh', '--manifest', str(manifest), '--key', candidate['key'],
             '--input', str(snapshot / candidate['upstream_sha'] / candidate['schema_path'])], work, log)
        second = {str(p.relative_to(work)): discovery.digest(p.read_bytes())
                  for folder in ('schemas', 'internal/generated')
                  for p in (work / folder).rglob('*') if p.is_file() and p.name != 'adaptation.lock'}
        require(first == second, 'official generation is nondeterministic')
        delta_report = log.with_suffix('.contract-delta.json')
        run([sys.executable, 'tools/contracts/capabilities.py', 'delta', '--baseline', str(baseline),
             '--candidate', str(work / 'schemas/wire-descriptors.json'),
             '--output', str(delta_report)], work, log)
        (work / 'schemas/contract-delta.json').write_bytes(delta_report.read_bytes())
        run(['make', 'test'], work, log)
        run(['go', 'vet', './...'], work, log)
        destination.mkdir(parents=True)
        run(['go', 'build', '-trimpath', '-o', str(destination / 'terraform-provider-routeros'), '.'], work, log)
        import shutil
        shutil.copytree(work / 'schemas', destination / 'schemas', ignore=shutil.ignore_patterns('adaptation.lock'))
        shutil.copytree(work / 'internal/generated', destination / 'internal/generated')
        shutil.copyfile(archive, destination / 'provider-source.tar')


def reconcile(root, output, *, sha=None, repository=None, extra=False, nightly=False,
              replay=False, force=False, producer=produce):
    output = output.resolve()
    require(not output.is_relative_to(root.resolve()) or output.is_relative_to(root.resolve() / '.local'),
            'output must be outside the source tree or under ignored .local')
    revision = tracked_revision(root)
    verification = verification_inputs(root)
    with lock(output):
        run_id = datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ') + '-' + uuid.uuid4().hex[:8]
        run_dir = output / 'runs' / run_id
        run_dir.mkdir(parents=True)
        summary = {'format': FORMAT, 'run_id': run_id, 'provider_revision': revision,
                   'verification_inputs_sha256': verification, 'replay': replay,
                   'compatibility': 'offline-only; no live-target certification',
                   'candidates': [], 'outcome': 'failure'}
        state_path = output / 'checkpoints.json'
        snapshot = run_dir / 'discovery'
        started = time.monotonic()
        try:
            command = [sys.executable, 'tools/schema/discover/discover.py', 'run',
                       '--output', str(snapshot), '--state', str(state_path)]
            for flag, enabled in (('--extra', extra), ('--nightly', nightly), ('--replay', replay), ('--force', force)):
                if enabled:
                    command.append(flag)
            if sha:
                command += ['--sha', sha]
            if repository:
                command += ['--repository-dir', str(Path(repository).resolve())]
            run(command, root, run_dir / 'discovery.log', timeout=600)
            manifest_path = snapshot / 'manifest.json'
            manifest = discovery.parse(manifest_path.read_bytes())
            summary.update(upstream_sha=manifest['upstream_sha'], deferred=manifest['deferred'])
            state = discovery.load_state(state_path)
            archive = run_dir / 'provider-source.tar'
            subprocess.run(['git', 'archive', '--format=tar', '--output', str(archive), revision],
                           cwd=root, check=True, timeout=60)
            for candidate in manifest['candidates']:
                item = {k: candidate[k] for k in ('key', 'version', 'channel', 'flavor', 'architecture', 'lane')}
                item.update(status='failed', live_tested=False, published=False)
                summary['candidates'].append(item)
                if reusable(output, candidate, state, verification) and not force:
                    item['status'] = 'unchanged'
                    continue
                # Force revalidates into a separate run directory; never overwrites a receipt/artifact.
                destination = run_dir / candidate['key']
                begin = time.monotonic()
                try:
                    producer(root, archive, candidate, manifest_path, snapshot, destination,
                             run_dir / (candidate['key'] + '.log'))
                except Exception:
                    delta = run_dir / (candidate['key'] + '.contract-delta.json')
                    if delta.is_file():
                        decision = discovery.parse(delta.read_bytes()).get('decision')
                        if decision == 'review-required':
                            item.update(status='review-required', review_artifact=str(delta.relative_to(output)))
                    raise
                proof = {'format': FORMAT, 'candidate_key': candidate['key'], 'stage': 'generated',
                         'success': True, 'provider_revision': revision,
                         'verification_inputs_sha256': verification,
                         'checks': ['official-generators', 'exact-specification', 'deterministic-generation',
                                    'reviewed-contract-delta',
                                    'tooling-tests', 'go-race-mock-tests', 'go-vet', 'go-build'],
                         'live_tested': False, 'published': False, 'artifacts': inventory(destination)}
                proof_raw = discovery.encode(proof)
                discovery.atomic(destination / 'generated-evidence.json', proof_raw)
                if 'generated' not in state['candidates'][candidate['key']]['stages']:
                    import shutil
                    durable = output / 'candidates' / candidate['key']
                    # Recover an artifact-only crash before checkpointing by replacing the unreceipted bundle.
                    if durable.exists():
                        shutil.rmtree(durable)
                    durable.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copytree(destination, durable)
                    with discovery.locked(state_path):
                        state = discovery.load_state(state_path)
                        discovery.advance(state, candidate['key'], 'generated', proof_raw)
                        discovery.atomic(state_path, discovery.encode(state))
                item.update(status='generated', seconds=round(time.monotonic() - begin, 3))
            statuses = [c['status'] for c in summary['candidates']]
            summary['outcome'] = ('pending' if manifest['deferred'] else
                                  'changed' if 'generated' in statuses else 'unchanged')
        except Exception as error:
            # No command output/connection credentials in summaries. Logs are offline only.
            summary['error'] = type(error).__name__
            raise
        finally:
            summary['seconds'] = round(time.monotonic() - started, 3)
            discovery.atomic(run_dir / 'summary.json', discovery.encode(summary))
            discovery.atomic(output / 'latest-summary.json', discovery.encode(summary))
        return summary


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, default=ROOT / '.local/maintenance')
    parser.add_argument('--sha')
    parser.add_argument('--repository-dir', type=Path)
    parser.add_argument('--extra', action='store_true')
    parser.add_argument('--nightly', action='store_true')
    parser.add_argument('--replay', action='store_true')
    parser.add_argument('--force', action='store_true')
    args = parser.parse_args()
    try:
        summary = reconcile(ROOT, args.output, sha=args.sha, repository=args.repository_dir,
                            extra=args.extra, nightly=args.nightly, replay=args.replay, force=args.force)
        print(discovery.encode(summary).decode(), end='')
        return 0 if summary['outcome'] != 'pending' else 2
    except (MaintenanceError, discovery.DiscoveryError, OSError, subprocess.SubprocessError) as error:
        print('Maintenance failed: ' + str(error), file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
