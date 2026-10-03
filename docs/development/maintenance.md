# Continuous provider maintenance — artifact-only milestone

This project is the Framework provider **and its ongoing restraml-driven maintenance loop**, not a one-off schema port. Coverage expansion extends the policies and lifecycles that this loop can safely maintain. Both official HashiCorp generators remain mandatory.

## Working baseline

Commit `1f522d3` establishes the sixteen-resource Framework/REST baseline. `schemas/baseline-manifest.json` records its immutable commit/tree, original tracked-file hashes, resource/field counts, producer pins and successful offline/live gates. Live evidence is restricted to RouterOS **7.24.5 x86_64/base CHR**; configuration lifecycle tests do not certify packet processing or SDK state migration. The guest, credentials and mutable disk were removed after the baseline rerun.

## Run from a clean committed provider checkout

```sh
make testmaintenance
python3 tools/maintenance/maintain.py

# Offline immutable upstream Git replay (committed blobs only).
python3 tools/maintenance/maintain.py \
  --repository-dir ../restraml \
  --sha adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a \
  --output .local/maintenance-replay

# Optional extra packages/nightly; stale nightly provenance fails unless explicitly replayed.
python3 tools/maintenance/maintain.py --extra --nightly
```

Requires Python 3.11+ with the backported tar extraction filters (3.11.8+), POSIX locks, Git, Bash, make and downloadable Go/official generators/Terraform. CI uses Python 3.14.7, Go 1.25.8 and Terraform 1.14.0. No upstream scripts or SDK callbacks are executed. The maintained consumer code must be committed and reviewed before running: isolated builds use a Git archive of exactly that revision, not an untracked/dirty workspace.

## Implemented loop

1. Resolve restraml main once (or validate an explicit SHA); discover stable/latest base, plus requested extra/nightly candidates. Existing freshness, path, metadata, version and hash gates apply unchanged.
2. Save exact upstream blobs and a provenance-qualified manifest. Persist discovered receipts only after the complete selected discovery passes validation.
3. For each candidate, extract the provider source archive into an isolated temporary workspace, adapt that exact snapshot and run **both official generators twice** with independent bundle/specification gates and deterministic output checks.
4. Run full Python tooling tests, Go race/unit/mock protocol/Terraform tests, vet and provider build with live flags/RouterOS credentials stripped from child environments.
5. Save schemas, generated models, source archive, binary, file hashes and generated-stage evidence. Advance **generated only**, after successful offline checks and artifact persistence. Never advance `tested` or `published`: no corresponding live target or publication is run by this command.
6. On a repeat with intact matching receipts/artifacts and verification inputs, report unchanged. Changed tests/orchestration/fixtures invalidate candidate keys alongside existing producer inputs. Force revalidates into a separate run directory without replacing any successful receipt/bundle. Partial unreceipted artifacts can be recovered; damaged successful bundles require restoring their original evidence, not silently blessing altered output.

`--output` contains `checkpoints.json`, `candidates/<key>/` immutable successful bundles, `runs/<run-id>/` immutable inputs/logs/attempt evidence, and `latest-summary.json`. Keep checkpoints **and their bundles together** when restoring. Output locks reject overlapping runs; checkpoint locks remain compatible with the discovery CLI. Discovery/build failures exit 1, requested missing artifacts exit 2 (pending, never a successful no-op). Previous good candidate outputs remain untouched, and failed summaries distinguish failures from unchanged inputs. An invalid discovery cannot advance its checkpoints. Discovery frontiers are distinct from generated/functional/publication stages.

## End-to-end validation

At provider revision `70015da`, committed restraml snapshot `adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a` selected stable 7.24.5/base and prerelease 7.25beta5/base. Both candidates passed two official-generation runs, all 59 Python tooling tests (45 existing + 14 maintenance), Go race/mock tests, vet and build, with no changes to the source checkout. Candidate generation/full offline verification took approximately 76 seconds each; a second run reused the intact evidence and reported unchanged. `schemas/maintenance-validation.json` records candidate keys, raw-input/evidence hashes, checks and measured timings. Only discovered/generated receipts were advanced: **the newer candidate has no live compatibility certification**.

Nineteen orchestration tests additionally exercise changed/unchanged inputs, force without replacing receipts, test-change invalidation, ignored-cache handling, dirty-checkout/output-path rejection, build/discovery failures, missing/tampered evidence, partial-artifact recovery, deferred artifacts, locking, documentation-only receipt reuse, retained review-required contract reports, a real producer delta rejection before tests/build, last-good preservation after review-failed force validation, policy/baseline key invalidation, and removal of live flags/credentials from offline subprocesses.

The producer now applies [`semantic-identity-v1`](capabilities.md) after deterministic official generation and before tests/build or receipt advancement. This compares adapted public contracts only; raw RouterOS semantics and new live lanes remain uncertified. Review-required candidates retain actionable delta reports in the run directory without altering last-good evidence. Workflow syntax/security lint passed with actionlint v1.7.7; hosted execution has not yet been observed.

## Scheduled workflow

`.github/workflows/maintenance.yml` runs daily at 07:30 UTC and by dispatch. Upstream inputs enter through environment variables/argument arrays, not interpolated shell code. It has read-only repository permissions, pinned actions/tools, bounded job/command runtimes and no signing, VM or router credentials. It never writes generated candidates into the source branch or performs releases. Failed-run evidence is retained too.

GitHub artifacts retain raw inputs, checkpoints, schemas/models, source archives, offline candidate binaries and summaries for **90 days**. Only the dedicated `.local/maintenance/` paths are uploaded; `.local/chr/` is never uploaded. These are **offline candidate artifacts**, not tested-latest releases. The initial implementation was pushed on 2026-10-03; hosted provider CI succeeded. Maintenance and new recovery-job conclusions are recorded separately in the [hosted checklist](hosted-checkpoint.md), not inferred from local passes.

### Verified recovery archives

`tools/maintenance/bundle.py` exports deterministic offline bundles, excluding runtime logs, unreceipted partial candidates and CHR credentials/disks. It retains checkpoints, successful candidate artifacts and exact discovery manifests/raw inputs. Export independently verifies candidate identities, snapshot/index/metadata hashes, checkpoint identities and generated evidence inventories. Existing archive names are never replaced.

```sh
python3 tools/maintenance/bundle.py export --output .local/maintenance \
  --archive .local/recovery/bundle.tar > .local/recovery/bundle.sha256
python3 tools/maintenance/bundle.py restore --output .local/restored \
  --archive .local/recovery/bundle.tar --sha256 '<trusted digest>'
```

Create `.local/recovery` before redirecting output. Restore requires a new directory and an externally trusted archive SHA256. It rejects traversal, links, duplicate members, oversized expansion, inventory corruption, missing snapshots and unsupported tested/published receipts. Verification happens in an isolated temporary directory before exposing checkpoints and bundles together. Restored generated proofs keep their original producer/verification bindings; subsequent fresh discovery must still pass current producer and verification-input gates. Hashes do not authenticate their own origin: a digest fetched from the same unauthenticated source is insufficient.

The maintenance workflow transfers this bundle to a separate runner for an independent restore plus fresh immutable-discovery/no-op check. This is a recovery demonstration, **not durable cross-run storage**: Actions artifacts still expire. Ten additional tests cover deterministic/immutable export, fresh restore/reuse, stale verification, missing/modified evidence and raw snapshots, unsupported stages, malicious tar members and tampering. See the [hosted checkpoint checklist and human-only decision boundary](hosted-checkpoint.md).

### Explicit remaining work

- Cross-run durable hosted receipt/bundle restoration (protected automation branch or permanent immutable storage). The current hosted workflow starts each fresh runner from an empty state and revalidates candidates; it does **not** pretend Actions caches or expiring artifacts are a permanent successful frontier. Local reuse works when the output directory is retained/restored. Add the hosted backend before claiming persistent scheduled deduplication or full A1 completion.
- Pinned SDK contract/test extraction and reconciliation, lifecycle capability classification, field-level semantic delta/promotion policy and reusable review candidates.
- Automated corresponding-version acceptance targets and live lane matrices; the fixed local 7.24.5 CHR baseline cannot certify a newly discovered version. Advance `tested` only with actual target evidence.
- Freshness/coverage ledger, outage/lag alerts and rolling review PR/issue handling; currently summaries expose observed SHA, versions, statuses, deferred artifacts and timings without inventing latest compatibility.
- Authorized nightly publication and protected stable Registry promotion. No release/signing gate is weakened by this artifact milestone.
