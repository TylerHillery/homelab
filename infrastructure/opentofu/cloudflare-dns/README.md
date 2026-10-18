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

Local state is the initial choice. It is ignored by Git and **not backed up by
OpenTofu**. A backup is deferred for the pilot; losing this file would require
recovering or importing managed records before safe future applies. Commit the
provider dependency lockfile. Never commit
state, plan files, API tokens, or generated credentials. Do not run applies
concurrently against local state. A remote backend can be evaluated later,
with a backed-up state migration rather than recreating resources.

Before adding resources, inventory the existing Cloudflare zone and import any
records this root will manage. Review a plan against the live zone; do not let
an initial apply replace existing records. The operator's narrowly scoped
Cloudflare token comes from the Bitwarden item
`homelab-cloudflare-dns-opentofu` through this project's `fnox.toml`, which
injects only `CLOUDFLARE_API_TOKEN` using the existing interactive
`mise run fnox:exec` wrapper. Do not put the token in `.tf` files or command
arguments. Run a read-only, live plan from this directory after unlocking
Bitwarden:

```sh
mise //:fnox:exec -- tofu plan -input=false
```

The initial apply added only those two records (0 changes, 0 destroys). Do not
apply future changes until the ingress, access policy, and rollback are reviewed.

DNS ownership is separate from ingress: a host's Caddy configuration will
map verified names to loopback services, while Ansible will eventually manage
Caddy's package, unit and configuration deployment. Framework WSL's Serve
TCP 443 route now forwards to its local Caddy listener; the host-specific
cutover and rollback are documented under `infrastructure/ingress/framework13-wsl/`.
