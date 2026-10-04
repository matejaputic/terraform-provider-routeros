# Disposable CHR on OrbStack and hosted runners

The harness supports pinned **7.24.5 and 7.25beta5 x86_64 CHR** on QEMU/TCG. The provider currently registers 152 resources, but `chr.py test` selects the preserved 125-constructor configuration suites, **not the 27 additional settings**. VETH is explicitly unavailable on base and tested with container. Focused additional-settings tests cover 19 base / 26 matching optional-package settings; physical LEDs are unavailable on CHR. See [coverage and evidence](coverage.md).

## Why QEMU, not an ISO machine

OrbStack hosts the unprivileged Linux QEMU container; it does not boot an arbitrary RouterOS ISO or create/change an OrbStack Linux machine. Apple Silicon uses software TCG, not nested KVM. The guest is an official preinstalled x86 CHR disk with pinned archive/disk hashes. An ARM64 installer/image is not the tested artifact; no ARM64 CHR acquisition/runtime recipe or live acceptance is implemented.

## Requirements

Use Docker context `orbstack`, Docker CLI, Python 3.11+, curl, downloadable Go 1.25.8 / Terraform 1.14.0, and internet for verified image/tool/package acquisition. The Docker base image, QEMU package version, CHR archive and raw disk are pinned. The default RouterOS recipe is 7.24.5; explicit versions must be selected consistently at startup and test.

Hosted `start --hosted` requires this owned GitHub repository, a GitHub-hosted runner and Docker's default context. Hosted Docker uses `linux/amd64`; local Apple Silicon uses `linux/arm64` for the container while the emulated RouterOS guest remains x86_64. Local startup refuses a different Docker context rather than changing it.

## Base configuration suites

Run from the repository root; cleanup must run on every outcome:

```sh
trap 'python3 tools/chr/chr.py stop' EXIT
python3 tools/chr/chr.py start --version 7.24.5
python3 tools/chr/chr.py test --version 7.24.5
```

Repeat with `--version 7.25beta5` on both commands for the second pinned recipe. The default selector is:

```text
^TestAcc(IPAddress|Collections|DHCPRouting|DHCPOptions|DHCPRelay|DNSRecord|Singletons|ExtendedCollections|Firewall|FirewallFamilies)CHR$
```

This exercises collections, DHCP options/relay, DNS records, nineteen settings, extended resources and ordered firewall families. It verifies disposable/version/architecture guards, create/read/update-or-replacement, imports, no-op plans, drift/deletion repair and owned-object cleanup as applicable. It does not select `TestAccAdditionalSettingsCHR` or certify actual packet/client/server exchanges.

## Matching optional-package lanes

```sh
# Enables matching container package; VETH configuration becomes available.
python3 tools/chr/chr.py start --version 7.24.5 --container
python3 tools/chr/chr.py test --version 7.24.5
python3 tools/chr/chr.py stop

# Broader owned-guest fixture for additional settings tests.
python3 tools/chr/chr.py start --version 7.24.5 \
  --packages container wireless user-manager
```

Both versions have hash-bound official package recipes. `container-recipes.json` retains the container-only bindings; `optional-package-recipes.json` binds container/wireless/user-manager members to the same reviewed archives. Upload uses bounded password SFTP to owned loopback SSH, one fixed reboot, then disables SSH and verifies the exact matching enabled package/version set. Provisioning files are removed even after failure. It does not change administrator keys, execute arbitrary scripts or start containers.

To run the focused settings suite against a started fixture without printing credentials:

```sh
trap 'python3 tools/chr/chr.py stop' EXIT
python3 - <<'PY'
import json, os, subprocess
from pathlib import Path
state = Path('.local/chr')
credentials = json.loads((state / 'credentials.json').read_text())
target = json.loads((state / 'target.json').read_text())
env = os.environ.copy()
env.update(ROS_HOSTURL=credentials['hosturl'],
           ROS_USERNAME=credentials['username'], ROS_PASSWORD=credentials['password'],
           ROS_TEST_DISPOSABLE='1', ROS_TEST_VERSION=target['version'],
           TF_ACC='1', TF_ACC_TERRAFORM_VERSION='1.14.0', GOTOOLCHAIN='go1.25.8')
subprocess.run(['go', 'test', './internal/provider',
                '-run', '^TestAccAdditionalSettingsCHR$', '-count=1',
                '-v', '-timeout=5m'], env=env, check=True)
PY
```

Base skips package-required settings; the broader package fixture makes 26 settings available. The physical LED resource remains unavailable on CHR. The direct focused invocation above does not write `chr.py test`'s acceptance receipt; record its source/target/package evidence separately and do not call it consolidated or maintenance-candidate certification. Database-path changes use explicit dependencies to avoid resetting concurrently configured user-manager settings.

## Isolation and cleanup

- Unprivileged QEMU container, capabilities dropped, no-new-privileges; 2-CPU/1536-MiB container limits and a 1-CPU/1024-MiB guest.
- REST `127.0.0.1:18780` and serial `127.0.0.1:18723`; optional package provisioning briefly publishes loopback SSH `18722`, then disables the guest service. No public/bridged host listener, privileged container, KVM or host networking.
- Restricted user networking blocks guest outbound connectivity. Its DHCP lease/gateway support replies to the forwarding peer; acquisition downloads occur on the host, not through unrestricted guest egress.
- A second virtio NIC named `tf-port` uses a QEMU hub with **no host/network backend**. A disconnected `tf-test` bridge supports IP tests. Acceptance uses disabled/disconnected/unattached owned objects and does not modify management connectivity.
- Serial bootstrap assigns a random admin password, enables isolated HTTP REST and disables other management services. Console input is CR only; buffers/passwords are never logged.
- `.local/chr/` is ignored and mode 0700; credentials are mode 0600. Never upload credentials, mutable disks or serial buffers. HTTP is appropriate only for this isolated fixture, not production credential transport.

`start` refuses a preexisting same-name container; startup failures clean the owned fixture. `stop` refuses a container without the ownership label, removes guest/credentials/mutable disk/provisioning files, and retains verified immutable download caches. Do not leave the privileged serial console available after testing. Existing containers/machines, Docker contexts and host routes/bridges are not changed.

## Hosted evidence boundary

`test.yml` currently has four version × base/container jobs on `ubuntu-24.04`, after offline verification. EXIT/signal traps and an always-run cleanup step remove the owned fixture; runner destruction is the final cancellation boundary. Only target provenance, successful acceptance receipts, whitelisted diagnostics and credential-free image-build logs are retained for 90 days.

Successful default receipts bind source commit/cleanliness and target/recipe/package hashes for the selected 125-constructor suites. Historical clean-source local/hosted four-lane runs are recorded in [configuration validation](../../schemas/resource-configuration-validation.json) and [hosted bindings](../../schemas/hosted-configuration-validation.json). Adding recipes/jobs, changing selector names or seeing offline success is not live evidence for a new candidate/version/architecture. Durable tested/published receipts and release publication remain unfinished.

Historical IP live findings remain regression-tested: literal `*` IDs are required in REST paths; `vrf` is read-only; bare-IP updates need an explicit host prefix. Those fixes do not authorize SDK-state migration or full reference parity.
