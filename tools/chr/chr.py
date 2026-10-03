#!/usr/bin/env python3
"""Disposable x86 CHR in QEMU/TCG on OrbStack or an owned GitHub-hosted runner.
No privileged container, bridged LAN, KVM, production-router inputs, or secret logs.
"""
import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import time
import urllib.error
import urllib.request
import zipfile

ROOT = Path(__file__).resolve().parents[2]
STATE = ROOT / '.local/chr'
NAME = 'routeros-provider-step1'
IMAGE = 'routeros-provider-chr:step1'
VERSION = '7.24.5'
RECIPES_PATH = ROOT / 'tools/chr/recipes.json'
RECIPES = json.loads(RECIPES_PATH.read_text())
CREDENTIALS = STATE / 'credentials.json'

def run(*args, **kwargs):
    return subprocess.run(args, check=True, cwd=ROOT, **kwargs)

def digest(path):
    with path.open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()

class Console:
    def __init__(self):
        deadline = time.monotonic() + 120
        self.buffer = b''
        self.events = set()
        while time.monotonic() < deadline:
            connection = None
            try:
                connection = socket.create_connection(('127.0.0.1', 18723), timeout=2)
                connection.settimeout(2)
                # Docker's host port can accept before QEMU's serial server is ready.
                # Require actual serial bytes, not merely an accepted TCP socket.
                connection.sendall(b'\r')
                initial = connection.recv(65536)
                if initial:
                    self.s, self.buffer = connection, initial
                    self.s.settimeout(1)
                    return
            except OSError:
                pass
            if connection is not None:
                connection.close()
            time.sleep(1)
        raise RuntimeError('CHR console did not become ready') from None

    def send(self, text):
        # CRLF submits twice on RouterOS's serial console, including passwords.
        self.s.sendall(text.encode() + b'\r')

    def expect(self, token, timeout=120):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if token in self.buffer:
                before, self.buffer = self.buffer.split(token, 1)
                return before
            try:
                chunk = self.s.recv(65536)
                if not chunk:
                    raise RuntimeError('CHR console disconnected')
                self.buffer += chunk
                for marker, event in ((b'Login:', 'login-prompt'), (b'Kernel panic', 'kernel-panic'),
                                      (b'Rebooting', 'guest-reboot'), (b'No bootable device', 'no-boot-device')):
                    if marker in self.buffer:
                        self.events.add(event)
            except TimeoutError:
                pass
        # Never print console buffers: they can contain secrets.
        raise RuntimeError('Timed out waiting for CHR console prompt')

    def command(self, command):
        self.send(command)
        response = self.expect(b'] > ', 30)
        if any(x in response.lower() for x in (b'failure:', b'bad command', b'syntax error', b'expected end')):
            raise RuntimeError('CHR bootstrap command failed: ' + command.split(' ')[0])


def request(credentials, path, method='GET', data=None):
    auth = base64.b64encode((credentials['username'] + ':' + credentials['password']).encode()).decode()
    r = urllib.request.Request(credentials['hosturl'] + '/rest' + path,
                               method=method,
                               data=json.dumps(data).encode() if data is not None else None,
                               headers={'Authorization': 'Basic ' + auth, 'Content-Type': 'application/json'})
    with urllib.request.urlopen(r, timeout=10) as response:
        body = response.read()
        return json.loads(body) if body else None


def execution_platform(hosted=False):
    context = subprocess.check_output(['docker', 'context', 'show'], text=True).strip()
    if hosted:
        if not (os.environ.get('GITHUB_ACTIONS') == 'true'
                and os.environ.get('RUNNER_ENVIRONMENT') == 'github-hosted'
                and os.environ.get('GITHUB_REPOSITORY') == 'matejaputic/terraform-provider-routeros'
                and context == 'default'):
            raise RuntimeError('Hosted execution requires the owned GitHub-hosted repository runner')
        return 'linux/amd64', 'GitHub-hosted Docker'
    if context != 'orbstack':
        raise RuntimeError('Select the OrbStack Docker context first; no other Docker engine will be modified')
    return 'linux/arm64', 'OrbStack Docker'


def validate_packages(packages, version=VERSION):
    if not isinstance(packages, list) or not packages:
        raise RuntimeError('CHR package inventory is missing or malformed')
    enabled = []
    for package in packages:
        if not isinstance(package, dict):
            raise RuntimeError('CHR package inventory is malformed')
        disabled = package.get('disabled', 'false')
        if type(disabled) not in (str, bool) or disabled not in ('false', 'no', False, 'true', 'yes', True):
            raise RuntimeError('CHR package enabled state is unknown')
        if disabled in ('false', 'no', False):
            enabled.append(package)
    if (len(enabled) != 1 or enabled[0].get('name') != 'routeros'
            or enabled[0].get('version') != version):
        raise RuntimeError('CHR enabled packages do not match the pinned base lane')


def install_container(credentials, version):
    # Only pinned official packages, an owned loopback guest, password SFTP and
    # a fixed reboot. No user SSH-key change or arbitrary RouterOS script.
    recipes_path = ROOT/'tools/chr/container-recipes.json'
    recipes = json.loads(recipes_path.read_text())
    if version not in recipes: raise RuntimeError('No reviewed container acquisition recipe')
    recipe = recipes[version]
    package = STATE/recipe['member']
    try:
        with urllib.request.urlopen(recipe['url'],timeout=120) as response:
            data=response.read(128*1024*1024+1)
        if len(data)>128*1024*1024 or hashlib.sha256(data).hexdigest()!=recipe['archive_sha256']:
            raise RuntimeError('container archive hash mismatch')
        import io
        with zipfile.ZipFile(io.BytesIO(data)) as archive: payload=archive.read(recipe['member'])
        if len(payload)!=recipe['package_bytes'] or hashlib.sha256(payload).hexdigest()!=recipe['package_sha256']:
            raise RuntimeError('container package hash mismatch')
        package.write_bytes(payload)
        console=Console();console.expect(b'] > ')
        console.command('/ip service set ssh disabled=no')
        console.s.close()
        env=os.environ.copy();env['GOTOOLCHAIN']='go1.25.8'
        result=subprocess.run(['go','run','tools/chr/package-upload/main.go',str(CREDENTIALS),str(package)],cwd=ROOT,env=env,timeout=120,capture_output=True)
        if result.returncode!=0:raise RuntimeError('owned container package upload failed; protocol output suppressed')
        console=Console();console.expect(b'] > ');console.send('/system reboot');console.expect(b'[y/N]:');console.send('y');console.s.close()
        deadline=time.monotonic()+60
        while subprocess.check_output(['docker','inspect','--format','{{.State.Running}}',NAME],text=True).strip()=='true':
            if time.monotonic()>deadline:raise RuntimeError('owned package reboot did not stop QEMU')
            time.sleep(1)
        run('docker','start',NAME,stdout=subprocess.DEVNULL)
        console=Console();console.expect(b'Login:');console.send('admin+ct');console.expect(b'Password:');console.send(credentials['password']);console.expect(b'] > ')
        console.command('/ip service disable ssh');console.s.close()
        deadline=time.monotonic()+60
        while True:
            try:request(credentials,'/system/resource');break
            except (OSError,urllib.error.URLError):
                if time.monotonic()>deadline:raise RuntimeError('extra guest REST unavailable') from None
                time.sleep(1)
        return dict(recipe,recipes_sha256=digest(recipes_path),installer='owned password SFTP; verified RouterOS package installation on reboot')
    finally: package.unlink(missing_ok=True)


def start(hosted=False, version=VERSION, container=False):
    if version not in RECIPES:
        raise RuntimeError('No pinned acquisition recipe for this RouterOS version')
    recipe = RECIPES[version]
    zip_hash, disk_hash = recipe['archive_sha256'], recipe['disk_sha256']
    url = f'https://download.mikrotik.com/routeros/{version}/chr-{version}.img.zip'
    platform, engine = execution_platform(hosted)
    existing = subprocess.check_output(['docker', 'ps', '-a', '--filter', 'name=^/' + NAME + '$', '--format', '{{.Names}}'], text=True).strip()
    if existing:
        raise RuntimeError('Disposable container already exists; run stop before starting a fresh fixture')
    STATE.mkdir(parents=True, exist_ok=True, mode=0o700)
    os.chmod(STATE, 0o700)
    # Never let an unsuccessful new attempt inherit a previous successful proof.
    (STATE / 'target.json').unlink(missing_ok=True)
    (STATE / 'acceptance-evidence.json').unlink(missing_ok=True)
    (STATE / 'diagnostics.json').unlink(missing_ok=True)
    archive = STATE / f'chr-{version}.img.zip'
    if not archive.exists():
        partial = archive.with_suffix('.download')
        try:
            # HTTP/2 GETs were reset by the vendor CDN in both local and hosted runs.
            # Prefer bounded HTTP/1.1 acquisition, never a different/unverified image.
            run('curl', '--http1.1', '-fsSL', '--connect-timeout', '15',
                '--retry', '2', '--retry-all-errors', '--retry-max-time', '300',
                '--max-time', '180', url, '-o', str(partial))
            if digest(partial) != zip_hash:
                raise RuntimeError('CHR downloaded archive hash mismatch; refusing cache promotion')
            partial.replace(archive)
        finally:
            partial.unlink(missing_ok=True)
    if digest(archive) != zip_hash:
        raise RuntimeError('CHR archive hash mismatch; refusing boot')
    original = STATE / f'chr-{version}.img'
    with zipfile.ZipFile(archive) as z:
        # Extract only the expected name, not arbitrary archive paths.
        with z.open(original.name) as source, original.open('wb') as target:
            shutil.copyfileobj(source, target)
    if digest(original) != disk_hash:
        raise RuntimeError('CHR disk hash mismatch; refusing boot')
    disk = STATE / 'test.img'
    shutil.copyfile(original, disk)
    os.chmod(disk, 0o666)  # unprivileged container UID; parent is mode 0700
    with (STATE / 'build.log').open('w') as log:
        run('docker', 'build', '--platform', platform, '-t', IMAGE, 'tools/chr', stdout=log, stderr=subprocess.STDOUT)
    credentials = {'hosturl': 'http://127.0.0.1:18780', 'username': 'admin', 'password': secrets.token_hex(24)}
    fd = os.open(CREDENTIALS, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, 'w') as f:
        json.dump(credentials, f)
    try:
        run('docker', 'run', '-d', '--name', NAME, '--label', 'routeros-provider.disposable=true',
        '--platform', platform, '--cpus', '2', '--memory', '1536m', '--cap-drop', 'ALL',
        '--security-opt', 'no-new-privileges', '-p', '127.0.0.1:18780:8080', '-p', '127.0.0.1:18723:2323',
        *(['-p','127.0.0.1:18722:8022'] if container else []),
        '-v', str(disk) + ':/chr.img', IMAGE, '-machine', 'pc', '-accel', 'tcg', '-m', '1024', '-smp', '1',
        '-drive', 'file=/chr.img,format=raw,if=ide', '-netdev', 'user,id=net0,restrict=on,hostfwd=tcp:0.0.0.0:8080-:80' + (',hostfwd=tcp:0.0.0.0:8022-:22' if container else ''),
        '-device', 'virtio-net-pci,netdev=net0',
        '-netdev', 'hubport,id=testnet,hubid=1', '-device', 'virtio-net-pci,netdev=testnet,mac=52:54:00:00:00:02',
        '-display', 'none', '-monitor', 'none',
        '-serial', 'tcp:0.0.0.0:2323,server=on,wait=off', '-no-reboot', stdout=subprocess.DEVNULL)
        console = Console()
        console.expect(b'Login:')
        console.send('admin+ct')
        console.expect(b'Password:')
        console.send('')
        console.expect(b'[Y/n]: ')
        console.send('n')
        console.expect(b'new password> ')
        console.send(credentials['password'])
        console.expect(b'repeat new password> ')
        console.send(credentials['password'])
        console.expect(b'] > ')
        # The default DHCP client supplies 10.0.2.15/24; restricted slirp omits
        # a default gateway, so add one for replies to Docker's hostfwd peer.
        for command in [
            '/ip route add dst-address=0.0.0.0/0 gateway=10.0.2.2',
            '/interface bridge add name=tf-test',
            '/interface ethernet set [find default-name=ether2] name=tf-port',
            '/ip service set www disabled=no available-from=0.0.0.0/0 port=80',
            '/ip service disable ftp', '/ip service disable ssh', '/ip service disable telnet',
            '/ip service disable winbox', '/ip service disable api', '/ip service disable api-ssl',
            '/ip service disable www-ssl', '/ip service disable reverse-proxy',
            '/system identity set name=terraform-acceptance-chr',
        ]:
            console.command(command)
        console.s.close()
        deadline = time.monotonic() + 60
        while True:
            try:
                target = request(credentials, '/system/resource')
                break
            except (OSError, urllib.error.URLError):
                if time.monotonic() > deadline:
                    raise RuntimeError('CHR REST did not become ready') from None
                time.sleep(1)
        provenance = {k: target.get(k) for k in ('version', 'architecture-name', 'board-name')}
        (STATE / 'target.json').write_text(json.dumps(provenance, indent=2) + '\n')
        if target.get('version') != recipe['target_version'] or target.get('architecture-name') != 'x86_64':
            raise RuntimeError('CHR version/architecture mismatch')
        packages = request(credentials, '/system/package')
        validate_packages(packages, version)
        if container:
            package_proof = install_container(credentials, version)
            packages = request(credentials,'/system/package')
            enabled=[p for p in packages if p.get('disabled') in ('false','no',False)]
            if {p.get('name') for p in enabled}!={'routeros','container'} or any(p.get('version')!=version for p in enabled):
                raise RuntimeError('extra guest package inventory mismatch')
            provenance['container_acquisition'] = package_proof
            provenance['package_flavor'] = 'routeros+container'
        else: provenance['package_flavor'] = 'base'
        provenance.update({'archive_sha256': zip_hash, 'disk_sha256': disk_hash,
                           'recipes_sha256': digest(RECIPES_PATH),
                           'accelerator': 'tcg', 'engine': engine,
                           'packages': [{k: p.get(k) for k in ('name', 'version', 'disabled')} for p in packages]})
        (STATE / 'target.json').write_text(json.dumps(provenance, indent=2) + '\n')
        print(f'Ready: disposable RouterOS {version} x86_64 at 127.0.0.1:18780 (credentials not displayed)')
    except BaseException:
        # Never save Docker logs or console buffers. These whitelisted fields
        # diagnose guest exit/OOM/readiness without disclosing bootstrap secrets.
        try:
            diagnostics = {'format': 'routeros-chr-diagnostics@1', 'version': version,
                           'console_events': sorted(console.events) if 'console' in locals() else []}
            result = subprocess.run(['docker', 'inspect', '--format', '{{json .State}}', NAME],
                                    capture_output=True, text=True, timeout=10)
            if result.returncode == 0:
                status = json.loads(result.stdout)
                diagnostics['container'] = {k: status.get(k) for k in ('Running', 'ExitCode', 'OOMKilled', 'Dead')}
            (STATE / 'diagnostics.json').write_text(json.dumps(diagnostics, indent=2) + '\n')
        except Exception:
            pass  # Diagnostic failures must never prevent guest/credential cleanup.
        finally:
            stop()
        raise


def test(version=VERSION):
    (STATE / 'acceptance-evidence.json').unlink(missing_ok=True)
    if version not in RECIPES:
        raise RuntimeError('No pinned test recipe for this RouterOS version')
    provenance = json.loads((STATE / 'target.json').read_text())
    if provenance.get('version') != RECIPES[version]['target_version']:
        raise RuntimeError('Acceptance recipe differs from the booted target')
    credentials = json.loads(CREDENTIALS.read_text())
    env = os.environ.copy()
    env.update(ROS_HOSTURL=credentials['hosturl'], ROS_USERNAME=credentials['username'], ROS_PASSWORD=credentials['password'],
               ROS_TEST_DISPOSABLE='1', ROS_TEST_VERSION=version, TF_ACC='1', TF_ACC_TERRAFORM_VERSION='1.14.0', GOTOOLCHAIN='go1.25.8')
    pattern = '^TestAcc(IPAddress|Collections|DHCPRouting|DHCPOptions|DHCPRelay|DNSRecord|Singletons|BatchACollections|Firewall|FirewallFamilies)CHR$'
    run('go', 'test', './internal/provider', '-run', pattern, '-count=1', '-v', '-timeout', '5m', env=env)
    evidence = {'format': 'routeros-chr-acceptance@1', 'success': True,
                'provider_revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
                'source_tree_clean': not subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT),
                'target_sha256': digest(STATE / 'target.json'), 'recipes_sha256': digest(RECIPES_PATH),
                'version': version, 'architecture': 'x86_64', 'flavor': provenance.get('package_flavor','base'),
                'test_pattern': pattern, 'terraform_version': '1.14.0',
                'limitations': 'configuration lifecycle only; no traffic processing or release binary verification'}
    (STATE / 'acceptance-evidence.json').write_text(json.dumps(evidence, indent=2) + '\n')


def stop():
    # Refuse to remove a same-name container not owned by this harness.
    p = subprocess.run(['docker', 'inspect', '--format', '{{index .Config.Labels "routeros-provider.disposable"}}', NAME], capture_output=True, text=True)
    if p.returncode == 0:
        if p.stdout.strip() != 'true':
            raise RuntimeError('Refusing cleanup of a container not owned by this harness')
        run('docker', 'rm', '-f', NAME, stdout=subprocess.DEVNULL)
    CREDENTIALS.unlink(missing_ok=True)
    for version in RECIPES: (STATE/f'container-{version}.npk').unlink(missing_ok=True)
    (STATE / 'test.img').unlink(missing_ok=True)
    print('Disposable CHR stopped; credentials and mutable test disk removed')

if __name__ == '__main__':
    import signal
    def cancel(signum,frame): raise KeyboardInterrupt()
    signal.signal(signal.SIGTERM,cancel)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['start', 'test', 'stop'])
    parser.add_argument('--hosted', action='store_true', help='Permit only this repository GitHub-hosted runner')
    parser.add_argument('--container', action='store_true', help='Install the matching pinned container package on the owned disposable guest')
    parser.add_argument('--version', choices=sorted(RECIPES), default=VERSION,
                        help='Pinned acquisition/acceptance recipe; defaults to baseline')
    args = parser.parse_args()
    if args.command == 'start':
        start(hosted=args.hosted, version=args.version, container=args.container)
    elif args.command == 'test':
        test(version=args.version)
    else:
        stop()
