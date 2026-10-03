# DHCP option wave: reviewed contract

Two additional canonical resources extend the original sixteen-resource baseline. No aliases/data sources, option sets/matchers/relays or DHCP client/server attachments are added in this wave.

Reference: read-only pinned SDK source `0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb` (MPL-2.0), `routeros/resource_ip_dhcp_client_option.go`, `resource_ip_dhcp_server_option.go`, related `_test.go` declarations and `mikrotik_crud.go`. Reference constructors/callbacks/tests were not executed. Immutable restraml input remains `adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a`, `docs/7.24.5/openapi.json`; the extracted fixture is reproduced from committed blobs.

| Canonical resource | Path | Reviewed fields |
| --- | --- | --- |
| `routeros_ip_dhcp_client_option` | `/ip/dhcp-client/option` | computed `id`, required `name`, required int64 `code`, required `value`, computed `raw_value` |
| `routeros_ip_dhcp_server_option` | `/ip/dhcp-server/option` | same core fields, optional+computed `comment` and boolean `force` |

## Semantics and intentional limits

- Maintain PUT collection, filtered GET by `.id`, PATCH/DELETE item, validated `*HEX` import, atomic refresh, failed-create recovery and ownership checks. No new action paths or singleton adapters.
- `code` is decimal on the wire, constrained to 1–254 (server's pinned validator, conservative client subset excluding reserved codes). Both code and value update in place. Names replace rather than rename dependent objects, consistent with the maintained named-collection boundary. PATCH never echoes name, ID or raw value.
- `value` is a required nonempty RouterOS option expression, sent unchanged. Unlike the SDK client subset's optional value, this preview requires an explicit value and exposes `raw_value` as read-only in both resources. No computed-value override is allowed. Literal `s'192.0.2.22'` and hex `0x01020304` are acceptance scenarios; no full expression grammar or variable/script semantics is certified. Remote rejection remains a diagnostic.
- `raw-value` aliases to computed string `raw_value`; it is never written. Import adopts RouterOS expressions/raw spelling. No speculative equivalence normalization hides actual value drift.
- Server comment omission reads as empty; explicit empty clears it. Server force omission reads as false, based on live probes; false is explicitly written as `no`, true as `yes`. Other missing configured fields fail closed rather than receive invented defaults.
- Client built-in options use **`default=true`**, not `builtin=true`, on 7.24.5. Reject default records on import/read and before update/delete, including refresh-disabled mutations. Malformed default flags fail closed. Generic existing built-in/dynamic guards remain intact.
- New fields are explicit catalog approvals, fed through the official OpenAPI/spec/Framework generators and independently exact output gates. No discovery-driven automatic exposure and no SDK state migration claim.

## Validation and isolation

Initial owned 7.24.5 x86_64/base probes confirmed string/hex value roundtrip, code update, computed raw bytes, force true/false and omitted false, and the default client-option ownership flag. Probe objects were deleted in `finally` blocks. No options were attached to a DHCP client, network or server; no DHCP exchange or traffic behavior is claimed.

Add typed validation/negative-read/ownership/recovery tests and real Terraform mock CRUD/import/empty-plan/update/clearing/replacement/drift tests. Add `TestAccDHCPOptionsCHR` to the explicit full-suite selector, using unique `tf-coverage-` names, independent state and exact disposable-version/architecture guards. Validate both pinned versions with all original regression suites. Records remain unreferenced, and the entire guest/credentials/mutable disk must be removed after every outcome.

## Completed validation

At clean revision `06d10a653af935cc99dc6635b968217c255909f7`, all six suites covering all eighteen registered resources passed locally on both pinned **7.24.5 and 7.25beta5 x86_64/base** guests. Both guests, credentials and mutable disks were removed. The DHCP option suite verifies literal-to-hex/code updates, raw readback, omitted false force, force true/false, comment clearing, both imports, empty second plans/applies, out-of-band server-option drift repair, deleted client-option recreation and both name replacements.

`make test` passed 110 Python tooling tests and Go race/mock coverage, plus provider/inspector vet, build, byte-identical repeated official generation and actionlint. The clean-source immutable maintenance replay generated stable/base and beta/base offline candidates and an intact repeat returned unchanged. Fail-closed delta tests remain intact against the explicitly expanded baseline. See [`schemas/dhcp-options-wave-validation.json`](../../schemas/dhcp-options-wave-validation.json).

Coverage is now **18 of 232 constructors**, with **214 constructors remaining** (aliases/data sources separately tracked). These are source-revision configuration lifecycle and offline-generation results, not live acceptance of a snapshot-generated maintenance candidate, a release binary or DHCP exchange. Hosted results for the final pushed revision must be observed separately.
