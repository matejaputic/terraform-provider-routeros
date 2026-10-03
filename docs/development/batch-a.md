# Batch A — implementation in progress, not a completed milestone

The owner authorized completion of the **fixed 107-constructor Batch A**. The
milestone target remains **125 total implemented constructors**. It has not been
reduced, completed, or substituted with a smaller wave.

**Current implementation accounting: 0/107 new public resources completed;
18 existing constructors remain registered.** No new resource schemas/models,
public registrations, live certification, maintenance-candidate certification or
release certification are claimed by this foundation checkpoint.

## Frozen ledger and immutable observations

[`schemas/batch-a-ledger.json`](../../schemas/batch-a-ledger.json) records all 107
IDs, explicit canonical-name proposals, deferred aliases, source locations,
static field candidates/unresolved declarations, triage risks, and path/method
observations for **7.24.5/base, 7.24.5/extra, 7.25beta5/base and 7.25beta5/extra**.
Its 107 complement IDs keep Batch B fixed. Pending statuses and empty evidence
are deliberate: inventory and observations do not authorize exposure.

Source baseline: `1897fb32c03c43fb48f6ab3020cc3baf2e2b57b2`; SDK reference:
`0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb`; published restraml input:
`adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a`. Every schema input and the exact
historical baseline matrix bytes are hash-bound. The archived matrix is distinct
from the current capability matrix, so later implementation status changes do
not shrink the denominator or invalidate the partition test in shallow CI.

```sh
python3 tools/contracts/batch_a.py --upstream-dir ../restraml
python3 -m unittest discover -s tools/contracts -p 'test_batch_a.py'
```

The builder reads committed blobs only, never upstream callbacks/scripts. It
rejects changed baseline bytes and refuses to overwrite differing ledger content.
Canonical names are **proposals**, not registrations or automatic aliases.

Observed shapes in **each base lane**: 86 full-CRUD collection candidates,
19 singleton candidates, one absent collection and one partial collection.
In **each extra lane**: 87 full-CRUD collection candidates, 19 singleton
candidates and one partial collection. Structural CRUD is not semantic approval.

Two cases specifically require maintained behavior rather than blanket CRUD:

- **`ResourceInterfaceVeth`:** absent from both pinned base publications, present
  with structural CRUD in both extra publications. Resolve published extra input,
  package/topology provisioning, runtime capability and lane-aware generation
  without fabricating base paths or silently weakening exact resource-set gates.
- **`ResourceUserSshKeys`:** no item PATCH in any of the four publications. The
  pinned `resource_system_user_sshkeys.go` declares required sensitive,
  replacement-owned `key`, plus user/comment and computed fingerprint metadata.
  Resolve replacement-only mutation policy and missing-key readback preservation;
  never PATCH an unsupported path, manage the acceptance admin's keys, or leak key
  material through errors/evidence.

Other unresolved rows stay inside Batch A. OSPF interface-template construction,
sets/nesting, secrets, ownership, async/import behavior and canonical wire values
still require explicit field/action contracts and tests, not a blanket approval.

## Maintained singleton lifecycle foundation

`internal/provider/singleton_resource.go` is **unregistered support**, not nineteen
implemented resources. Its explicit lifecycle follows static reference
`resource_actions.go:311–355` and `resource_actions_default_system.go`:

- Read the singleton object through GET at its fixed policy path.
- Create-as-update and update POST the fixed `/set` action. Never PUT a collection,
  PATCH an item, transmit `.id`, or write computed fields.
- Identity derives from the path (`/system/identity` → `system.identity`). Import
  rejects item IDs and other singleton identities before any network operation.
- Destroy only relinquishes Terraform management, retaining device settings and
  warning the user. It requires no connection and never resets/deletes settings.
- Preflight reads precede writes. A failed create write/read retains the known
  baseline and deterministic identity for recovery. Malformed/unauthorized/404
  reads preserve state and report errors rather than silently removing a global
  settings resource.
- Primitive string/boolean/int64 payload and atomic readback are tested. Omitted
  writable fields stay omitted, explicit false becomes `no`, empty note clears,
  multiline notes remain supported, and unknown mutations/control characters are
  rejected. Collection replacement/default/conditional/array policies are rejected
  until their singleton equivalents have reviewed implementations.

Test-only schemas and a test-only provider exercise the engine; they do **not**
replace the two required official generators. Before exposing each of the nineteen
singletons, add its reviewed validators/defaults/field semantics, adapter and
exact generated-file/spec gates, official generated schema/model binding, and
applicable live evidence. Runtime registration remains unchanged meanwhile.

## Amortized immutable-input adaptation

`SchemaSnapshot` in `tools/schema/normalize/normalize.py` parses and validates the
same immutable input once per CLI invocation and observes inventory once. Every
resource still independently passes the existing exact lifecycle/field/codec/
replacement/ownership-policy checks. Different input hashes cannot reuse a cache;
report classifications are copied and cannot mutate sibling observations.
Secondary per-resource reports omit an unused full inventory copy; the final
bundle still contains the complete observed inventory and exact reviewed set.

The cached CLI produced byte-identical current normalized OpenAPI, generator
configuration, wire descriptors and upstream input compared with the existing
artifacts. No generated model or public schema changed. A **single local
18-policy fixture sample** measured 0.098s without reuse versus 0.013s with reuse
(7.59×); this saves only 0.085s on that fixture and is **not** a milestone timing
estimate or a substitute for larger batching and actual implementation work.

## Targeted checks actually observed

- `go test ./internal/provider -run '^TestSingleton' -count=1 -race`: passed,
  including a real Terraform CLI mock create/update/import, empty plans, drift
  repair, destroy-retains-settings, malformed refresh and recovery cases.
- Normalizer tests: **26 passed** (including five new snapshot-isolation/gate tests).
- Contract tooling tests: **31 passed** (including five planning-ledger tests).
- Cached CLI adaptation bundle verification and four current artifact byte
  comparisons: passed.
- `go vet ./internal/provider` and `git diff --check`: passed.

No new CHR guests, credentials, hosted runs or pushes were created for this
foundation work. No full 125-resource regression or final maintenance replay has
occurred. Existing eighteen-resource historical evidence is unchanged.

## Resume the same milestone

1. Integrate reviewed singleton contracts through both official generators and
   exact independent gates; keep `/set` runtime versus observed item PATCH
   distinctions explicit. Do not expose the shared helper without those bindings.
2. Execute **all** 88 collection rows by family with useful reviewed fields and
   maintained validators/codecs/ownership/readback/recovery. Resolve VETH and
   SSH-key special cases inside the original roster.
3. Add per-resource real mock/negative scenarios and safe applicable target lanes;
   retain all eighteen regressions. Amortize fixtures/CLI sessions and use bounded
   measured suite timeouts, not test exclusions.
4. At completed integration, refresh current—not historical—policy/provenance/
   capability bindings and perform the one consolidated official deterministic
   generation/full offline/two-version live/immutable replay/hosted delivery gate.
5. Only then report 107/107 complete, 125 implemented and 107 Batch B remaining.
