# State storage bootstrap

This **separate, locally stated** OpenTofu root manages the private R2 bucket
`homelab-tofu-state`, created with one apply and verified with a no-change
second apply. `cloudflare-dns` uses the bucket for its remote state. This root
cannot store its own state in a bucket it creates. Keep its local state backed up
outside Git, and retain a recovery path for the bucket if this machine is lost.
`prevent_destroy` guards planned deletion but does not protect against manual
deletion, losing the local state, or removing the resource from configuration.

## Bootstrap and recovery

The account ID and bucket name are non-secret values in `main.tf`. Keep public
access and development URLs disabled. If rebuilding this root after state loss,
**import** the existing bucket rather than creating it a second time. The
account-scoped provisioning token is stored in the personal Bitwarden item
`homelab-cloudflare-r2-bootstrap` and is distinct from the bucket-scoped S3 key
and the zone-scoped DNS token.

The bootstrap root's local state has a snapshot in a personal Bitwarden Secure
Note; verify it can be restored before relying on it for recovery. The original
DNS state was copied and verified outside the checkout before migration; retain
that copy. R2 has no S3-style bucket object versioning, so ongoing independent
versioned backups and a tested restore remain follow-ups.

Run the static checks without credentials:

```sh
mise //infrastructure/opentofu/state-bootstrap:check
```

From this directory, after unlocking Bitwarden, review a live plan before
each future apply. Do not apply before reviewing the account ID, bucket name,
and proposed actions:

```sh
mise //:fnox:exec -- tofu plan -input=false
mise //:fnox:exec -- tofu apply -input=false
```

If importing an existing bucket, follow the provider's import ID format
`ACCOUNT_ID/BUCKET_NAME/JURISDICTION` and verify a subsequent plan before an
apply. Keep local state, generated plans, and `.terraform/` out of Git.

## Remote state for other roots

The DNS root uses **R2 S3 credentials** with `Object Read & Write` restricted to
this bucket. They are different from the provider token. Its `fnox.toml`
supplies them as backend environment variables, not `-backend-config` arguments
or committed files. Migration used `tofu init -migrate-state -force-copy` only
after confirming the destination was empty; resource data matched the local
snapshot and a live plan showed no changes. OpenTofu assigned a new remote
lineage during the copy. Do not repeat `-force-copy` against populated state.
Test locking deliberately before concurrent applies. R2 alone is not a
versioned backup.

See [Infrastructure as Code Layout](../../../docs/architecture/infrastructure-as-code.md)
for the provider-independent root boundary rules.
