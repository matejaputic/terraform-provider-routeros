# Hosted maintenance/release checkpoint

The current provider has **152 registered resources** (106 collections / 46 settings), zero data sources and 80 unimplemented reference constructors. Release gates must certify the actual reviewed candidate, not a hard-coded historical count. Current `.github/workflows/test.yml` performs full offline verification and four 7.24.5/7.25beta5 × base/container CHR jobs; the CHR selector covers the preserved 125-constructor suites, not the 27 additional settings.

Recorded [expanded hosted bindings](../../schemas/hosted-configuration-validation.json) at `48d6b43` certify the historical 125-constructor configuration-tested set, including target/package hashes and cleanup. The [additional settings evidence](../../schemas/additional-settings-validation.json) is focused dirty-source development evidence: 19 base / 26 matching optional-package settings, with physical LEDs unavailable. No consolidated current 152-resource hosted or snapshot-candidate live certification is recorded. Do not infer hosted outcomes merely from a push or workflow definition.

The repository settings and run observations below are historical observations from 2026-10-03, not a fresh audit of branch protection, credentials or environments.

## Decision boundary

An **exception** is a decision that absolutely requires the owner's authority or judgment. Ordinary discovery, generator, test, CI, storage, provisioning or delivery failures are operational work: diagnose, repair and retry autonomously within the approved scope. Do not turn every failed check into an approval request.

Only irreducible decisions enter one digest-bound packet, with a concrete recommended resolution, executable patch/configuration where applicable, alternatives and actual validation. Examples include changing reviewed public semantics, broadening authorized scope, acquiring protected credentials or accepting unresolved compatibility trade-offs. Missing infrastructure remains a blocker; approval does not substitute for evidence. Owner notification receipt requires an owner acknowledgement, not an inferred success from creating a PR.

## Prerequisites observed

On 2026-10-03, using authenticated `gh` against the existing repository:

- Origin remains `git@github.com:matejaputic/terraform-provider-routeros.git`; default branch is `main`.
- Actions are enabled. Default workflow permissions are read-only; Actions cannot approve PR reviews.
- `main` has no branch protection; no repository environments exist. No protections were removed or settings changed.
- Existing seven local commits were pushed normally, advancing `main` from `fad8dc6` to `9ed9f52`.
- [Provider CI run 37125567754](https://github.com/matejaputic/terraform-provider-routeros/actions/runs/37125567754) succeeded, including exact unchanged official generation, tooling, race/mock tests, vet/build and example formatting.
- [Immutable replay run 37125584264](https://github.com/matejaputic/terraform-provider-routeros/actions/runs/37125584264) succeeded with publication SHA `adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a`. Downloaded summaries report generated (not live-tested/published) 7.24.5/base and 7.25beta5/base candidates.
- [Recovery run 37125989726](https://github.com/matejaputic/terraform-provider-routeros/actions/runs/37125989726), provider revision `b55d751`, succeeded in both `candidates` and independent `recovery-proof` jobs. Fresh-runner restore and fresh immutable discovery returned unchanged, without rebuilding or publishing.

## Implementation checklist

- [x] Verify hosted authentication/repository identity and push existing safe CI.
- [x] Observe successful hosted provider CI on the pushed revision.
- [x] Implement deterministic offline recovery archives and fail-closed fresh-directory restoration.
- [x] Observe fresh-runner recovery/no-op job on the new revision.
- [ ] Select and implement protected durable checkpoint index plus immutable bundle storage. The current Actions transfer is a recovery test, not permanent storage. Suggested backend: protected automation branch containing hash-bound pointers to permanent immutable release/storage objects. Authenticate pointers, serialize updates and never silently replace an existing digest.
- [x] Generalize pinned CHR acquisition/runner execution and prove the historical 125-constructor source suites on pinned 7.24.5/7.25beta5 x86_64 base/container guests (base VETH explicitly unavailable).
- [ ] Add the additional settings to consolidated clean-source/hosted acceptance with matching package lanes; do not substitute schema/mock or unavailable-hardware results for live evidence.
- [ ] Bind live acceptance to the actual snapshot-generated maintenance candidate and advance independently verified tested receipts only after that gate.
- [ ] Implement authenticated digest-bound decision packet and automatic continuation for human-only exceptions; ordinary failures stay autonomous.
- [ ] Verify owner notification routing with an acknowledged test. Personal email/mobile/OS settings cannot be assumed or changed silently.
- [ ] Deliver uniquely versioned provider nightlies only after corresponding-target acceptance; independently download/verify assets and durable receipts.
- [ ] Demonstrate transition, failed-change recovery and last-good preservation on hosted runners.

## Observed autonomous recovery

- `37126454596`: offline CI passed; baseline acquisition failed on a vendor HTTP/2 connection reset before startup. Cleanup succeeded. Bounded HTTP/1.1 acquisition resolved the observed reset and verifies the pinned hash before cache promotion.
- `37126951245`: acquisition and pinned QEMU build passed; the early serial socket reset. Cleanup succeeded. Readiness now requires actual serial bytes rather than merely an accepted Docker proxy socket.
- `37127449183`: both guests were running and reached login; the diagnostics loop incorrectly shadowed the requested prompt. A new regression test guards the corrected loop. The observed beta REST version is exactly `7.25beta5`, not an assumed channel-suffixed spelling.
- `37128095056` at clean revision `93d3527`: offline CI and both full live lanes passed. Both downloaded target/evidence bindings were independently verified. Both complete suites also passed locally from that clean revision, and all guests/credentials/mutable disks were removed. See [`schemas/hosted-chr-validation.json`](../../schemas/hosted-chr-validation.json).

These failures were ordinary operational problems, not human-only exceptions. Source-revision live acceptance does not automatically certify generated maintenance candidates, released binaries or other lanes. These historical runs did not authorize new resource/field/action exposure or Registry promotion. Subsequent resource waves require their own explicit reviewed contracts and evidence. Successful offline restoration is not a provider release.
