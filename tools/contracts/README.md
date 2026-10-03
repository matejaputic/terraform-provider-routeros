# Pinned SDK reference contract extraction

Consumer-owned **static inspection**, not SDK execution or a Framework emitter. The reference is read-only and fixed to `0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb`. This is behavioral seed infrastructure for continuously maintaining the Framework provider against new immutable restraml publications; reference schemas do not replace that ongoing input feed.

From the provider repository root:

```sh
make testcontracts
python3 tools/contracts/extract.py --repository-dir ../terraform-provider-routeros
# Explicit reviewed artifact refresh; never edits reference source/public runtime schemas.
python3 tools/contracts/extract.py \
  --repository-dir ../terraform-provider-routeros --output schemas/reference-contracts
```

Requires Git, Python 3.11+ (POSIX locks) and Go 1.25.8. No SDK imports or additional dependencies are used by the Go AST inspector. Go compiles only the maintained consumer inspector; upstream initializers, constructors, Configure/CRUD/import/upgrades, defaults and tests **never execute**. Committed Git blobs are consumed through `ls-tree`/`cat-file` at the full pin, irrespective of dirty reference working files. A different pin requires an explicit reviewed tooling update.

## Bundle

- `reference-contracts.json`: source SHA/file SHA256s, extractor hashes/tool pin, resource/data-source registration and alias maps, constructor schema declarations, helper declarations, source/function/test indexes and unresolved-shape findings.
- `reconciliation.json`: all sixteen current descriptor contracts, per-field static declaration comparisons, unexposed reference fields, source locations and explicit review status. Binds the descriptor and policy hashes. **Existing reviewed overlays remain authoritative.**
- `manifest.json`: hash-bound final marker written last under an output lock; `extract.verify_bundle()` rejects changed/missing/partial artifacts. Refresh the bundle through the extractor, never edit generated evidence by hand.
- `NOTICE.md`: MPL-2.0 reference attribution (maintained, not an extractor output).

Actual pinned counts: **256 resource names / 232 distinct resource constructors, 16 data-source names, 209 indexed test functions**. Data-source constructors overlap resource constructors and are counted separately from resource implementation work. The bundle was generated twice byte-identically. All sixteen current resource contracts reconcile to source declarations; IP comment/disabled optional+computed and computed-only VRF differences remain explicit, as do name replacement and other approved Framework deviations. Synthetic Terraform `id` is not confused with SDK service metadata. No registration/public-schema/runtime change is made.

## What is recovered versus unresolved

The inspector resolves scalar constants/string concatenation, literal slices, global literal schema helpers, primitive type expressions, required/optional/computed/default/sensitive/replacement properties, descriptions and declarative call arguments. Source expressions retain validators, diff suppression, nested element definitions, importer/state-version/lifecycle bindings and path/ID/unset/transform metadata. Static enum/default arguments are captured without invoking their functions. Functions/closures and SDK behavior remain **needs-review**, with declaration source provenance—not assumed Framework equivalents. Test functions and conventional related test files are indexed, not yet adapted.

A constructor with one literal map and no detected indexed/append/delete mutations is labelled `static-declarations-only`: this **does not mean complete semantics or eligibility for exposure**. Multiple maps, unresolved keys and detected mutations produce review findings. Nested maps remain in their owning field's expression, never flattened into top-level attributes. Nonliteral helper-generated maps can be unresolved; nested SDK structure, conditional mutations, global initialization changes, version-dependent behavior and arbitrary helper semantics are not interpreted. Static review can miss indirect mutation, so every constructor carries a separate semantic review requirement. The full pinned inventory has 240 declaration-only and 7 unresolved constructor shapes across the combined resource/data-source index.

Reconciliation mechanically compares declaration types, modes, replacement and sensitivity where statically known. It records unknown flags as unknown, never as false. Defaults, codecs, equivalence, ordering/ownership, sensitive readback, wire transformations and migrations require reviewed semantic helper/policy mapping and tests; a clean declaration comparison cannot approve them. List/set/map semantics and nested shapes are not interchangeable merely because an OpenAPI outer type matches. Excluded fields are findings, not automatic generation candidates.

Checkpoint validation at provider revision `45381e3`: 73 Python tooling tests (14 extractor tests), provider race/mock checks, inspector/provider vet, build, unchanged official schema generation and actionlint v1.7.7 passed. The maintenance loop was rerun end-to-end against committed restraml `adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a`: stable/base and beta/base candidates passed full offline checks, then a repeat reported unchanged. Evidence is in `schemas/reference-checkpoint-validation.json`. Only discovered/generated receipts exist; no hosted run or newer live lane is certified.

Next: resolve high-value helper/contract gaps into a capability matrix; add versioned delta/promotion policies and adapt indexed reference scenarios by family. Approved overlays then feed normalized OpenAPI → official OpenAPI generator → independent contract gate → official Framework generator. This extractor **never emits Framework code** and does not claim full E1 semantic completion or new live compatibility.
