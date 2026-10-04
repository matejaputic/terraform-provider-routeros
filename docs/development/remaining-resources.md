# Remaining resource support

The remaining constructor identities are recorded in [the frozen 107-constructor plan](../../schemas/remaining-resource-plan.json). This is an identity/partition record, **not an exposure authorization**. The eighteen original constructors and verified 125-constructor baseline remain preserved.

## Current checkpoint

- **27 reviewed settings constructors** generated through the pinned official OpenAPI and Framework generators and registered: 152 public constructors in the current working tree.
- **80 remaining constructors are not integrated**. Complete constructor coverage, release publication, and maintenance-candidate live certification are not claimed.
- Maintained singleton lifecycle: GET object, path identity/import, fixed POST `/set`, atomic malformed refresh, mutation failure recovery, and destroy by unmanagement only. Destroy does not reset global settings.
- Explicit scalar/CSV contracts and sensitivity: `tools/schema/normalize/additional_settings_policy.py`, `internal/catalog/additional-settings.json`; registry references only in `internal/provider/additional_settings_bindings.go`.
- All 27 have schema/validation/atomic-failure/secret tests and Terraform mock create/no-op/update/import/drift/explicit-clearing/destroy coverage. The full offline/race suite passes (80.8% provider statement coverage).
- Focused **dirty-source** live configuration lifecycles passed on 7.24.5 and 7.25beta5: 19 constructors on base; 26 with matching enabled container, wireless, and user-manager packages. Physical LED controls report unsupported on CHR and are not live-certified.
- Live evidence is focused development evidence, **not** clean-source consolidated acceptance, a released binary, or a generated maintenance candidate. Hosted additional-settings verification and fresh expanded maintenance replay/no-op remain pending.

## Reviewed live findings

- IP cloud accepts `ddns-enabled=auto/yes`; `no` is rejected by these pinned targets.
- Disk `auto-media-interface` is an interface name, not the symbolic interface list `static`; the disconnected owned bridge `tf-test` is used.
- SMB interfaces likewise use `tf-test`, with SMB disabled.
- Legacy CAPsMAN manager changes keep certificate requirements disabled and use reviewed upgrade policy changes; no automatic certificate generation is performed.
- User manager message authentication uses `no/yes-access-request`.
- **Changing the user-manager database path reloads its database and can reset user-manager settings and advanced options.** Manage the database first and make dependent user-manager settings depend explicitly on `routeros_user_manager_database`. Do not relocate a production database casually. The owned disposable fixtures use this dependency; database contents and migration compatibility are not certified.
- Physical LED controls require an applicable physical board. CHR GET returns an empty settings object and attempts at non-default control report unsupported hardware; no mock result is represented as physical-board evidence.

## Package lane safety

`tools/chr/optional-package-recipes.json` binds each package member's size/hash to the already reviewed official archives for both versions. Only container, wireless, and user-manager packages may be provisioned on the owned loopback guest. The installer uses bounded password SFTP, one fixed reboot, disables SSH afterwards, verifies the exact enabled package/version set, and removes provisioning files. It does not change administrator keys, execute arbitrary scripts, or run containers.

## Remaining work

Review and implement the other 80 constructor contracts, including ordinary collections, structured/custom lifecycles, hardware-dependent resources, local cryptographic state, and asynchronous operations. Preserve the publication, independent code-spec, exact generated-file-set, ownership, and maintenance delta gates. Hardware and package absence must remain individually documented rather than being treated as passing live acceptance. Refresh metadata and evidence as subsequent subsets become reviewed; the frozen ledger must not be used to authorize exposure.
