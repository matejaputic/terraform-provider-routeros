# routeros_system_user_sshkeys

<!-- Rendered by tools/docs/resources.py from reviewed contracts; edit resource_notes.json for caveats. -->

REST path: `/user/ssh-keys`. Reviewed Terraform Plugin Framework resource; unlisted fields/actions and SDK-state migration are not supported.

## Lifecycle

Creates with PUT and reads by internal ID; changes require replacement rather than item PATCH. Destroy deletes the owned collection item.

Reviewed ownership checks can reject built-in, dynamic or otherwise unmanageable objects. Import is not blanket permission to modify every row. Failed refresh preserves prior state; a known allocated ID is retained for create recovery.

## Attributes

| Attribute | Type | Mode | Notes |
| --- | --- | --- | --- |
| `bits` | int64 | Computed | — |
| `fingerprint` | string | Computed | — |
| `id` | string | Computed | — |
| `key` | string | Required | Change requires replacement; Sensitive; protect Terraform state; Preserves known secret on omitted/masked readback |
| `key_owner` | string | Computed | Sensitive; protect Terraform state |
| `key_type` | string | Computed | — |
| `user` | string | Required | Change requires replacement |

Computed attributes and IDs are never mutation inputs.

Terraform sensitivity redacts normal display; it does **not** encrypt state. Use a protected state backend. Secret preservation is per-field, not a generic substitution for any masked value.

## Import

```sh
terraform import routeros_system_user_sshkeys.example '*A'
```

Use an internal `*HEX` ID, not a name or numeric row index. Import does not imply compatibility with historical SDK state.

## Resource-specific notes

- Mutation is replacement-only; item PATCH is unsupported. Only reviewed ssh-ed25519, ssh-rsa and ecdsa-sha2-nistp256 public keys without authorized-key options are accepted. SHA256 fingerprint ownership is verified, including RouterOS padding.
- Administrator and authenticated-user keys cannot be managed. Import cannot recover public-key material; supplying a key after fresh import requires replacement. Omitted/masked key material preserves known state, while unrelated fingerprints are rejected.

## Verification scope

This resource belongs to the preserved 125-constructor configuration-tested set: clean-source local/hosted suites cover 7.24.5 and 7.25beta5 x86_64/base and matching container-package lanes, with VETH explicitly unavailable on base. Evidence binds historical source revisions, not a released binary or generated maintenance candidate.

See [coverage and evidence](../development/coverage.md) for exact boundaries. Configuration acceptance does not certify packet processing, arbitrary script/container execution, external services, hardware availability, release publication or SDK-state migration.

For a starter topology, see the [README usage example](../../README.md) and [maintained examples](../../examples). Choose unused interfaces/subnets and protect management connectivity.
