# Disposable CHR on OrbStack (Apple Silicon)

Coverage update: `test` runs IP-address, six-resource networking and four-resource DHCP/static-routing and isolated firewall/address-list/order acceptance. An additional virtio NIC named `tf-port` connects only to a QEMU hub without a host/network backend, permitting bridge-port tests without touching the management NIC. Existing disconnected `tf-test` bridge remains for IP tests. No additional ports, privileges or egress are introduced. See [coverage evidence](coverage.md).

## Why QEMU, not an ISO machine

The supplied OrbStack docs describe supported Linux distributions, not arbitrary ISO boot. Its x86 Linux support uses Rosetta for userspace; this does not boot an x86 RouterOS kernel. The supplied docs also rule out nested KVM on Apple Silicon.

The VirtualBox CHR instructions translate to QEMU: attach the official CHR disk image, allocate 1 GiB RAM, attach a virtio network adapter, boot, and set the initial admin password. We use OrbStack's Docker engine as the Linux QEMU host and explicitly select software TCG, not KVM. No OrbStack Linux machine is created or changed.

The suggested ARM64 ISO URL and the ARM64 CHR image both returned HTTP 200, but neither was used or acceptance-tested. An ARM64 installer ISO is not the same artifact as a preinstalled CHR disk. The supplied RouterOS documentation describes x86 CHR; the tested target is explicitly **x86_64 CHR 7.24.5 stable base**, not ARM64 or the nightly train.

## Requirements and commands

OrbStack Docker context `orbstack`, Docker CLI, Python 3.11+, curl, Go 1.25.8 (downloadable by Go), and internet for first-time image/tool downloads. Observed engine: OrbStack 2.2.3. The container base is pinned by digest, QEMU by Debian package version, and CHR ZIP/raw disk by SHA256. Package availability failures are visible; the harness never silently upgrades QEMU.

Run from the repository root:

```sh
python3 tools/chr/chr.py start
python3 tools/chr/chr.py test
python3 tools/chr/chr.py stop
```

Always run `stop` after testing, including after a failed test. Startup failures clean up the owned container automatically. `start` refuses a preexisting same-name container; `stop` refuses one lacking the harness ownership label. Only this container, its mutable disk, and its local credentials are removed. Cached verified image downloads remain for replay. No existing containers/machines, OrbStack defaults, Docker contexts, host routes or bridges are changed.

## Isolation and bootstrap

- Unprivileged QEMU container, all capabilities dropped, no-new-privileges, 2-CPU/1536-MiB container limit; guest has 1 CPU and 1024 MiB RAM.
- Docker publishes REST `127.0.0.1:18780` and serial console `127.0.0.1:18723`. No bridged LAN or publicly bound host listener.
- QEMU user networking uses `restrict=on`: guest outbound connectivity is blocked. The default DHCP lease supplies `10.0.2.15/24`; an explicit gateway route is needed for replies to Docker's forwarding peer.
- Serial bootstrap changes the blank admin password to a random value, enables HTTP REST, disables other configurable management services and creates a disconnected `tf-test` bridge. Acceptance mutates only an IP on this bridge, not the management interface.
- The private guest's www service accepts forwarded peers; protection is the explicit loopback host mapping and restricted QEMU network, not production-grade HTTPS. HTTP would be unsuitable for a production management network.
- Console input uses **CR only**, because CRLF submits passwords twice on RouterOS. Buffers and passwords are never printed.
- `.local/chr/` is ignored and mode 0700; `credentials.json` is mode 0600. Do not upload it or the mutable VM disk. The console is privileged access even without a password prompt: leave the container running only while testing.

## Verified behavior and fixes

`TestAccIPAddressCHR` checks actual target version/architecture before mutation. Terraform 1.14.0 drives protocol 6 create/read/update/import/destroy, repeated empty plans/applies, out-of-band deletion/recreation, unique IDs, booleans and computed fields. A final REST query verifies no test addresses remain.

Live testing exposed regressions absent from the original mock:

1. `%2A3` in REST resource paths produces HTTP 400; literal `*3` succeeds. The client preserves literal `*` but escapes other special characters as a single path segment. Mock regression checks now assert the encoded path too.
2. `vrf` is present on read but rejected on write in 7.24.5 (`unknown parameter vrf`). It is computed-only for the supported slice. Mutations use configured values, not computed plan/state defaults. This is a documented exception to the reference SDK schema, not a claim of full migration compatibility.

3. Updating a bare IP can inherit the previous netmask. The client sends an explicit host prefix and retains the configured bare representation only on semantically identical readback. A live update/empty-plan regression test covers this.

The live run does not establish ARM64, extra-package, prerelease/nightly, older SDK-state migration, native API, or Registry release compatibility. Those retain their separate plan gates. Signing/publication remains disabled.
