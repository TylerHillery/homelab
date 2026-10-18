# Framework WSL ingress

`Caddyfile` routes OpenCode (`127.0.0.1:4096`) and HLIMS staging
(`127.0.0.1:8081`) on `127.0.0.1:8444`. Tailnet-only Tailscale Serve forwards
TCP 443 to that listener; public DNS points at this machine's Tailscale IP
without Cloudflare proxying. OPNsense Unbound allows tailnet answers for
`homelab.tylertries.com`. **No Funnel or public listener.** All current
tailnet devices can reach both sites; names and certificates are public.

The opt-in Ansible Caddy role installs and enables `caddy-ingress.service` for
user-session startup. Its fnox runner fetches the DNS-01 token from Bitwarden
at each start using `~/.config/homelab/fnox.local.toml` (encrypted session).
If the session expires, run `mise run bw:unlock` before restarting the unit.
WSL restart behavior has not been verified; `Linger=no` means the unit cannot
run when the user manager is stopped. See the [shared role](../caddy/README.md)
for setup and the unattended-restart limitation.

```sh
mise //infrastructure/ingress/framework13-wsl:check
systemctl --user status caddy-ingress.service
tailscale serve status --json
```

Verify both HTTPS names from another tailnet device with `curl` and normal
certificate checks (no `-k`). Changing the Tailscale IP requires a reviewed DNS
update. Stopping the unit makes both names unavailable; Serve is configured
separately and remains in place.

## Optional follow-ups

Investigate whether Caddy should bind directly to Framework WSL's Tailscale
IP on port 443 instead of using Serve's raw TCP forward to `127.0.0.1:8444`;
review privileged-port permissions, Tailscale startup/IP changes, credential
recovery, and a tested rollback before changing the working path.
Serve is part of the existing Tailscale daemon, not an extra process; Caddy
still supplies custom-domain TLS and routes by hostname either way.
Separately, investigate a LAN fallback through Windows-to-WSL reachability,
restricted firewall rules, and local DNS for the same HTTPS names.
HLIMS has no built-in login, so define trusted LAN clients or proxy
authentication before opening any LAN path; neither change narrows the
tailnet's current allow-all policy on its own.
