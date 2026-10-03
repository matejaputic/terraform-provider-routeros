# Lifecycle triage and reviewed contract delta gate

The next extraction/automation checkpoint adds two deliberately separate artifacts:

* [`schemas/capability-matrix.json`](../../schemas/capability-matrix.json) accounts for all **256 resource names / 232 resource constructors**, preserving aliases, source references and related reference test files. Its callback-pattern families are **146 ordinary collection candidates, 53 singleton candidates, 1 hardware update-only, 32 custom/unresolved**. These are static triage categories, not eligibility or implementation certification. Hardware semantics hidden behind custom helpers remain custom/unresolved; absence of a risk flag does not certify safety.
* [`schemas/maintenance-policy.json`](../../schemas/maintenance-policy.json) defines `semantic-identity-v1`. Maintenance candidates must retain the approved public contract after normalization. A [`routeros-contract-delta@1`](../../tools/contracts/capabilities.py) report records identity-keyed JSON-pointer findings and baseline/candidate/policy hashes. This is **public contract identity**, not a proof that upstream RouterOS semantics are unchanged.

## Matrix evidence and helper mapping

The classifier consumes and verifies the pinned reference bundle, current descriptor hash and exact current policy hashes. It checks the eighteen implemented subsets' paths against source declarations and maps each exposed field to maintained codec helpers, defaults/read defaults, conditional readback, type/mode, sensitivity and replacement rules. IP address-specific boolean/string helpers remain separate from shared collection helpers. Helper source hashes bind the mapping to maintained implementations. No callback is imported or executed; no Framework code is emitted.

Singleton, update-only/custom callbacks, unresolved schema construction/path, nested/set fields, sensitive declarations, SDK state upgrades and firewall ordering/action applicability are retained as review risks. SDK state upgrades are **not** Framework migrations. All rows independently require semantic review and explicitly forbid automatic exposure. Default/validator/diff-suppression equivalence and capability gaps still need reviewed semantic work and adapted scenarios. The current eighteen implementations remain reviewed subsets; the matrix creates no new resources or field parity claims.

```sh
python3 tools/contracts/capabilities.py matrix --output schemas/capability-matrix.json
make testcontracts
```

The committed matrix is checked against deterministic recomputation by the tooling tests. MPL-2.0 reference attribution is in [`schemas/reference-contracts/NOTICE.md`](../../schemas/reference-contracts/NOTICE.md).

## Automatic maintenance boundary

The producer archives committed provider source and saves that revision's approved descriptors **before** invoking the official generators. After two exact, deterministic generation passes and **before** tests/build/receipt advancement, it assesses the adapted candidate descriptors against the reviewed policy-bound baseline.

Only descriptive field provenance and identity-list ordering are ignored. Resource/field additions or removals, defaults, validators, primitive or nested contract data, codecs, wire names, method/path/ID changes, type/mode, sensitivity, replacement, schema version, migration claims, policy revisions and previously unknown keys require review. Duplicate JSON keys/identities, nonfinite JSON, malformed contracts and an unbound baseline fail closed. No broad whitelist of primitive fields authorizes automatic exposure.

```sh
python3 tools/contracts/capabilities.py delta \
  --baseline schemas/wire-descriptors.json \
  --candidate /path/to/candidate/wire-descriptors.json \
  --output .local/contract-delta.json
```

Exit 0: `offline-compatible`; exit 2: `review-required` with an actionable report; exit 1: malformed/unbound evidence. Reports identify paths/kinds, not potentially sensitive field values. A compatible result **does not** advance `tested` or `published`, authorize a release, or certify a new version/package/architecture.

On a review decision the orchestrator stops without advancing a generated receipt, retains `<key>.contract-delta.json` outside the disposable build workspace and exposes its relative path in the run summary. Last-good bundles/receipts are untouched. Successful reports are copied into candidate schema evidence and hash-bound by `generated-evidence.json`. Policy/baseline changes invalidate discovery and offline-verification reuse. Existing review failures can recover after a reviewed source/policy change; GitHub issue/PR creation and permanent hosted frontier restoration are still unfinished.

## Checkpoint verification

Implementation committed at `17b7c0e`. All **90 Python tooling tests** pass (26 contract/classifier tests, 19 orchestration tests, 45 existing schema/discovery tests), along with provider race/mock checks, inspector/provider vet, build, unchanged official generation and actionlint v1.7.7. Two complete matrix recomputations are byte-identical. A full restraml `adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a` replay produced both 7.24.5/base and 7.25beta5/base candidates with `reviewed-contract-delta` evidence and empty identity-delta findings; repeat execution verified intact bundles and returned unchanged. Hashes and limits are retained in [`schemas/capability-checkpoint-validation.json`](../../schemas/capability-checkpoint-validation.json). No live acceptance was rerun or newly certified.

The gate operates on **adapted public contracts**, not complete raw upstream schemas. Unexposed raw fields remain unexposed; a new upstream specification alone cannot certify corresponding-target compatibility or unchanged packet-processing semantics. Raw semantic delta analysis, finer reviewed adaptation rules, matching-version live acceptance, durable hosted restore, review delivery and nightly/stable publication remain later milestones.
