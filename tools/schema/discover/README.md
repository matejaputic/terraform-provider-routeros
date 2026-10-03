# Deterministic restraml discovery (Step 2)

Run from the provider repository root. Requires Python 3.11+ and POSIX file locking; CI pins Python **3.14.7**, the locally tested version. No additional Python dependencies or upstream scripts are executed.

```sh
# Resolve tikoci/restraml main ONCE, then fetch only full-SHA raw URLs.
python3 tools/schema/discover/discover.py run

# Optional independent extra packages, bounded historical backfill, nightly slot.
python3 tools/schema/discover/discover.py run --extra --backfill 2 --nightly

# Explicit immutable replay. Full lowercase 40-character commit SHA only.
python3 tools/schema/discover/discover.py run --sha adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a --extra

# Read committed blobs from a trusted local restraml mirror, never working files.
python3 tools/schema/discover/discover.py run --repository-dir ../restraml --nightly --replay

make testdiscovery
```

Default outputs are ignored `.local/discovery/manifest.json`, unmodified raw blobs under `.local/discovery/<SHA>/docs/`, and `.local/discovery-checkpoints.json`. `--output` and `--state` allow isolated replay runs. Local Git mode assumes the operator supplied a trusted mirror of `tikoci/restraml`; it does not authenticate a mirror's origin. Network mode fixes the repository and hostnames in consumer-owned code.

## Selection and validation

- Accept only `restraml-docs-index@1`, `docs` root, safe exact index paths, unique version directories and allowlisted artifact filenames. RouterOS 7 stable, beta, RC and `_ab` version forms are parsed numerically; upstream strings never become shell source.
- Stable is **latestStableVersion**; the latest versioned lane is **latestVersion**, even if another indexed RC sorts higher. Identical pointers produce one candidate per flavor. Backfill selects up to eight other numerically ordered indexed versions; at most 24 candidates including nightly/extras. Historical backfill does not lower or trip current-lane frontiers.
- Fetch actual base/extra `openapi.json` independently. A 404 defers that candidate, regardless of index/latest pointers; dispatch success is not consulted. Missing index is a failure. Indexed semantic metadata missing from the immutable commit is an inconsistent snapshot and fails.
- Parse bounded JSON with duplicate keys/nonfinite numbers rejected; validate OpenAPI envelope/path/operation shape, nonempty operations and exact `info.version`. Full reference resolution, compositions and Terraform semantic curation remain Step 3.
- Canonical schema architecture is explicitly **x86**, based on the observed upstream publication contract. Arm64 inspect data does not imply an arm64 OpenAPI or acceptance result.
- Nightly additionally requires `nightly.json`, matching index version/publication time, canonical x86 phase provenance, and agreement of **every embedded x86/arm64 base/extra nightlyVersion and postVersion**, even when extras are not selected. Selected inspect `_meta.version`/architecture must match too.
- Aggregate and phase publication must be no older than **seven days**; future aggregate dates beyond five minutes fail. Current stable/latest/nightly frontiers cannot regress in numeric version, and nightly publication time cannot regress.
- `--replay` explicitly permits stale/regressing historical data, but never mixed versions, invalid JSON/path/schema or inconsistent timestamps. Replay does not move current-lane frontiers. `--force` reruns pending generation/testing decisions but does **not** bypass freshness or overwrite publication receipts.

## Candidate and dedup contract

Each candidate records repository, immutable upstream SHA, channel, RouterOS version, base/extra flavor, canonical architecture, schema path/raw SHA256, raw and semantic inspect/provenance hashes, publication metadata, processing-input fingerprint, key and separate stage status.

Keys bind raw schema bytes, semantic metadata, policy/runtime/template revisions, generator pins, Go/dependency locks and maintained runtime/normalizer/generation/discovery source hashes. Volatile metadata `generatedAt`, `builtAt`, boot/enrichment durations, the index timestamp and upstream commit identity are audit data, not key inputs. Thus a timestamp-only publication commit does not invent new work, but an overwritten same-version schema, semantic metadata change, policy/runtime change or tool pin change does.

The first implementation reports:

| Outcome | Meaning |
| --- | --- |
| `changed` | At least one candidate still needs successful generation, or force was requested. A previously discovered but failed/unattempted generation is retryable work. |
| `unchanged` | Every selected candidate already has a successful generation receipt and nothing is deferred. Later stages remain visible separately and may still be pending. |
| `not-yet-published` | No generation work remains, but one or more requested artifacts are absent. Check the deferred list even in mixed `changed` results. |
| `discovery-failure` (exit 1) | Outage, invalid selected schema, mixed/stale/regressing data, corrupt checkpoint or unsafe input. Never a successful no-op. |

Network requests retry at most three times (20-second timeout, 1/2-second backoff), with a 300-second overall fetch deadline and 64-MiB per-artifact bound. Redirects and nonretryable HTTP errors fail visibly; 404 is not retried. Git reads have 30-second subprocess bounds and blob-size checks. Immutable cached blobs cannot be replaced with different bytes.

## Successful checkpoints only

Discovery records **discovered** only, after the entire selected snapshot passes validation. It does not generate, test or publish provider code. Step 3's [snapshot adapter](../normalize/README.md) now consumes this manifest with input/metadata/key/producer verification; generation receipts still require trusted orchestration evidence. Status sequence is `discovered → generated → tested → published`; later steps must supply a matching successful evidence file:

```json
{"candidate_key":"<64-character key>","stage":"generated","success":true}
```

```sh
python3 tools/schema/discover/discover.py checkpoint \
  --key '<candidate key>' --stage generated --evidence '<successful stage evidence.json>'
```

The trusted stage caller is responsible for actually running/validating its stage; this command is a receipt protocol, **not** a substitute for generation, acceptance or release authorization. Evidence is hashed, stages cannot skip predecessors, failed evidence cannot advance, identical receipts are idempotent and successful receipts cannot be replaced by different evidence. A force rerun may produce new external diagnostics but retains an existing successful receipt. Publication replacement is forbidden.

POSIX locks serialize discovery/checkpoint writers; atomic fsync/rename protects checkpoints. A malformed selected snapshot preserves the prior good manifest/checkpoints. Partial artifact-write recovery may leave verified raw cache files, but never an advanced stage. For hosted workflows, carry this file in a protected automation branch or immutable manifests (Step 6), **not** an Actions cache. Repository locks alone are not cross-runner publication locks.

## Verification

18 offline tests cover stable/beta/RC/nightly parsing, pointers/backfill, immutable Git reads despite dirty working files, missing base/extra/provenance, malformed/unsafe/mixed inputs, staleness/replay, metadata/tool-policy dedup, outage retries, successful stage ordering/idempotency, failure preservation and concurrent checkpoint writers. Fixtures are small synthetic JSON, regenerated with `python3 tools/schema/discover/fixtures/build.py`.

Real network discovery resolved `adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a` once and selected stable 7.24.5/base and latest 7.25beta5/base. Local immutable replay with one backfill, extras and nightly yielded eight candidates. Its August nightly provenance is correctly rejected without explicit replay, preserving prior successful discovery. These are **discovered**, not automatically generated/tested/published candidates; Step 1 acceptance does not certify all discovered lanes.
