# Hosted sixteen-resource checkpoint

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
- [Immutable replay run 37125584264](https://github.com/matejaputic/terraform-provider-routeros/actions/runs/37125584264) was dispatched with publication SHA `adc39cdbb0a3062aa93cc7aff50ef1185a1c0b0a`. Its conclusion and assets must be inspected before recording success.

## Implementation checklist

- [x] Verify hosted authentication/repository identity and push existing safe CI.
- [x] Observe successful hosted provider CI on the pushed revision.
- [x] Implement deterministic offline recovery archives and fail-closed fresh-directory restoration.
- [ ] Observe fresh-runner recovery/no-op job on the new revision.
- [ ] Select and implement protected durable checkpoint index plus immutable bundle storage. The current Actions transfer is a recovery test, not permanent storage. Suggested backend: protected automation branch containing hash-bound pointers to permanent immutable release/storage objects. Authenticate pointers, serialize updates and never silently replace an existing digest.
- [ ] Generalize pinned CHR acquisition and runner execution, then prove corresponding-version full sixteen-resource acceptance. Hosted QEMU/TCG avoids assuming KVM; actual runner resources, image availability, enabled packages and bounded readiness still need verification.
- [ ] Implement authenticated digest-bound decision packet and automatic continuation for human-only exceptions; ordinary failures stay autonomous.
- [ ] Verify owner notification routing with an acknowledged test. Personal email/mobile/OS settings cannot be assumed or changed silently.
- [ ] Deliver uniquely versioned provider nightlies only after corresponding-target acceptance; independently download/verify assets and durable receipts.
- [ ] Demonstrate transition, failed-change recovery and last-good preservation on hosted runners.

No new resource, field, action, target lane or Registry promotion is authorized by these changes. Successful offline restoration is not live compatibility or a provider release.
