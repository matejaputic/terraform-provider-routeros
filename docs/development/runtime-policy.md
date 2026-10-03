# Maintained runtime policy (Step 4)

Step 4 baseline revision: `framework-rest-preview-v3`. Superseded by `framework-rest-preview-v7` with [sixteen-resource networking/DHCP/routing/firewall coverage](coverage.md); the IP-specific guarantees below still apply. Step 4 originally registered only `routeros_ip_address`; no data sources, native API transport, singleton/hardware/order/async/action lifecycles or SDK-state migration claim.

## REST and configuration

Each configured provider instance owns its transport, credentials, RouterOS version and sorted enabled-package inventory. Configure reads `/system/resource` and `/system/package` once; resources reuse that client. No global RouterOS version or saved configuration context. Every request uses its current operation context; cancellation/deadline identity survives through a redacted error wrapper. Failed Configure closes idle connections and shares no client.

Only `http`/`https` are accepted. `api`/`apis`, embedded credentials, query/fragment URLs and malformed hosts produce diagnostics, not panics. TLS verification is enabled, minimum TLS 1.2; insecure is explicit opt-in. CA input is a validated PEM file; CA and insecure conflict. Redirects are not followed. Config/environment priority and explicit-empty behavior remain unchanged.

Paths/query parameters are structurally encoded. IDs are one escaped segment, except literal `*` required by RouterOS. Resource/import IDs must match `*` plus hexadecimal digits; mutations cannot target the collection with an empty ID. Path traversal/query injection controls are rejected. Request/response bodies and credentials are never logged. Response reads check I/O errors and a 4 MiB bound. Non-2xx results are typed status errors without body contents. Empty/204 is accepted for no-output mutations; empty/null/malformed JSON fails when an object/array is required, never masquerading as deletion.

## Version, packages and drift

Consumer-owned parsing supports stable patch versions, beta/RC numeric sequences and `_ab` nightlies, including a RouterOS channel suffix. Beta precedes RC then stable; numeric sequences sort numerically. Nightlies are ordered only within the same nightly base. No fictitious nightly-versus-stable or cross-base nightly ordering is used for policy.

Only the RouterOS 7 REST family is implemented. Unknown syntax/other majors fail Configure; versions other than reviewed 7.24.5 warn as experimental. This is a family/capability guard, **not** a claimed supported version range. Even matching version does not certify a different architecture/package lane.

Actual enabled packages must include `routeros` and match system version. Disabled packages do not confer capabilities and may have older versions. Missing/duplicate/invalid/mixed metadata fails Configure. The IP-address resource is built-in; extra packages do not expose additional resources. Accounts now need read access to both system resource and package metadata. Extra-package installation/provisioning and extra-lane acceptance remain deferred.

Reviewed IP wire aliases remain `.id` → `id`, `actual-interface` → `actual_interface`, with bool yes/no writes and true/false/yes/no reads. Inspection of the pinned reference drift table found **no IP-address rename**; do not apply unrelated container/routing/interface renames or invent speculative version thresholds. Missing required address/interface or malformed field types fail refresh without changing state. New version-specific renames require an explicit reviewed policy and regression/live evidence; broad drift dispatch is not ported.

## Framework lifecycle/state

Payloads come from configuration, not computed plan/state. Null/unknown optional values are omitted; explicit empty comment and false disabled are sent. Computed fields/IDs/VRF are never echoed. Bare IP writes use explicit host prefixes, retaining equivalent bare representations on read. Real prefix/address drift stays visible.

Read continues filtered collection GET with `.id` for out-of-band deletion. Only typed 404 or an actual empty array means absent; auth/transport/JSON errors do not remove state. Refresh decodes into a temporary model and commits only after validation, preventing partial state corruption. DELETE 404 is idempotent. Create stores its ID and known configured values before follow-up read, converting unresolved computed values to null so a failed read has recoverable, fully known partial state.

Provider password remains sensitive; masked password fields in remote system metadata never overwrite connection credentials (concurrent-alias regression coverage). **No implemented resource has a sensitive read-back field.** Generic masked resource-secret round-tripping is not certified or exposed; future sensitive resources need their own preservation/consistency tests, not blind star-string substitution.

Schema version is still 0. Framework import is maintained passthrough of a validated internal ID. There is no SDK state migration/UpgradeState implementation because no old-state compatibility is claimed; require migration fixtures before adding one or suggesting `terraform state replace-provider`.

## Evidence

Offline race tests cover concurrent Configure aliases with different versions/credentials, package capabilities and mismatches, beta/RC/nightly ordering, verified/custom-CA/insecure TLS, redirect protection, operation deadlines/cancellation, response-size/read failures, unsafe paths, strict string/bool decoding, atomic failed refresh, null/unknown/empty payloads, import validation, typed idempotent delete and ID recovery after failed create-read. Existing protocol/schema/catalog/password-sensitivity and Terraform 1.14.0 mock lifecycle gates still pass.

Live disposable CHR **7.24.5 x86_64/base** was rerun successfully after these changes: create/read/update/import/delete, typed computed/boolean state, empty second plans/applies, out-of-band deletion/recreation, bare-address update consistency, and Configure package discovery. Credentials, container and mutable disk were removed afterward. No beta/nightly/ARM64/extra lane or historical-state compatibility was newly certified.

Reference behavioral guides remain pinned to `0d8c069c20a012300dfeeb96cb343ad7a5e7ebfb` (MPL-2.0); no SDK callbacks/global state/body logging were copied. Reference updates must be explicit reviewed Conventional Commit changes/PRs, with provenance refresh, regeneration and regression gates; nightly generation must not fetch latest reference code.
