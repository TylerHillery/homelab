# Cloudflare DNS

This OpenTofu root owns shared `tylertries.com` DNS records for the homelab.
Its first two records point Framework WSL hostnames at the observed
Tailscale IPv4 address, with Cloudflare proxying **off**. They were applied
after the custom-domain Caddy ingress and certificates were tested from a
tailnet device. Review the address before each apply or after node
re-enrollment. Do not use a separate state for each application that
adds a record in this zone; keep zone-wide records in this one root. Other
independently managed infrastructure gets its own root and state.

```sh
mise install --monorepo
mise //infrastructure/opentofu/cloudflare-dns:check
```

The check runs `tofu fmt -check`, provider initialization in a separate,
backend-disabled `.terraform/check` data directory, `tofu validate`, and
TFLint's Terraform rules. The ignored directory reuses the installed provider
on subsequent runs, rather than downloading it for every commit.
The same check runs from hk when OpenTofu configuration or its tool
configuration changes. It does not plan, apply, migrate state, or validate the
R2 connection, and needs no credentials or access to the live backend.

State was migrated from local storage to the private R2 S3 backend at
`homelab-tofu-state/cloudflare-dns/terraform.tfstate`. The migrated R2 object
was checked against the pre-migration local backup (same resource data), and a
live plan showed no changes. OpenTofu assigned a new lineage while copying to
the remote backend; the original lineage remains in the backup. The working
state is now **remote**: a local `terraform.tfstate` left by migration is not
the active state or a reliable backup. Commit the provider dependency lockfile,
but never commit state, plan files, API tokens, or credentials.

The separate [`state-bootstrap`](../state-bootstrap/) root created the private
R2 bucket. A copy of the original DNS state was verified at a private path
outside the checkout before migration; retain it for recovery. The bucket-
scoped R2 S3 key is different from the DNS provider token; `fnox.toml` supplies
both only to the invoked command. Never put keys in `-backend-config` or
tracked files. R2 has no S3 bucket object versioning, so add independent,
versioned backups and a tested restore before relying on R2 alone. The
configured S3 lockfile was acquired and released during migration; continue
to avoid concurrent applies until locking has been deliberately tested.
See [Cloudflare's R2 backend guide](https://developers.cloudflare.com/terraform/advanced-topics/remote-backend/).

Before adding resources, inventory the existing Cloudflare zone and import any
records this root will manage. Review a plan against the live zone; do not let
an initial apply replace existing records. The zone-scoped DNS token and the
bucket-scoped R2 S3 credentials come from separate Bitwarden items through
this project's `fnox.toml`. Do not put secrets in `.tf` files or command
arguments. Run a read-only live plan from this directory before each apply:

```sh
mise //:fnox:exec -- tofu plan -input=false
```

The initial apply added only those two records (0 changes, 0 destroys). Review
future plans against the live ingress and access policy before applying.

DNS ownership is separate from ingress: the opt-in Ansible Caddy role installs
the binary, user unit and host Caddyfile, which route verified names to loopback
services. Framework WSL's Tailscale Serve still forwards tailnet TCP 443 to
Caddy's loopback TLS listener; see
`infrastructure/ingress/framework13-wsl/` for the current configuration.
