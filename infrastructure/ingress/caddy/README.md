# Caddy ingress

The shared build supplies Caddy with Cloudflare DNS-01. The opt-in Ansible
`caddy-ingress.yml` playbook installs it with each host's Caddyfile and fnox
provider references; it does **not** manage Tailscale Serve or backends.

```sh
mise //infrastructure/ingress/caddy:build
mise //infrastructure/ingress/caddy:check
mise //infrastructure/ansible:caddy:preview framework13-wsl
mise //infrastructure/ansible:caddy:apply framework13-wsl # after reviewing the preview
```

The shared `check` builds Caddy first; the explicit `build` command is only
needed when preparing the binary for Ansible deployment without running checks.

For each host, provide `infrastructure/ingress/HOST/{Caddyfile,fnox.toml}`;
the latter contains a **Bitwarden reference, never a token**. Install the pinned
fnox and Bitwarden CLIs and an age-encrypted session at
`~/.config/homelab/fnox.local.toml` for the service user before applying.
The role starts the user systemd unit and restarts it when managed files change;
systemd handles it between applies. It requires a loopback-only Caddy listener.
Add new hosts to the inventory and the tasks' allowed host choices before use.

fnox retrieves the token at service start; Caddy holds it in its process
environment. No plaintext token file is installed. A valid Bitwarden session
is required for **each** start or restart, so this is not unattended recovery.
Never reuse the Framework user's vault session on other hosts. See the
[Framework host notes](../framework13-wsl/README.md) for the current route.
