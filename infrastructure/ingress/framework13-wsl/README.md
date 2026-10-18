# Framework WSL ingress pilot

This per-host Caddyfile routes OpenCode and HLIMS staging by hostname without
path rewrites. Both backends remain on loopback, and the pilot listener is
**127.0.0.1:8444 only**. On Framework WSL the pilot binary is installed at
`~/.local/bin/caddy-homelab`, and `caddy-pilot.service` is active and **enabled
for user-session startup**. It obtained valid certificates for both hostnames
using Cloudflare DNS-01. The DNS-only A records now point to the Framework's
Tailscale IP; Tailscale Serve forwards tailnet TCP 443 to Caddy's loopback TLS
listener. Both backends returned HTTP 200 with verified TLS from Buck over the
tailnet. The previous `.ts.net` HTTPS Serve route has been replaced.

OPNsense Unbound initially filtered the Tailscale `100.x` A records as DNS
rebinding. The Ansible-managed Unbound **Private Domains** exception for only
`homelab.tylertries.com` now permits the answers without disabling rebinding
protection globally; see `opnsense/playbooks/site.yml`. Buck resolves and
reaches both ordinary HTTPS URLs with verified TLS.

Build the pinned Caddy version with the pinned Cloudflare DNS plugin and check
the Caddyfile (no server or certificate request starts):

```sh
mise //infrastructure/ingress/framework13-wsl:build
mise //infrastructure/ingress/framework13-wsl:check
```

The binary is generated under ignored `.build/`; copy it to
`~/.local/bin/caddy-homelab` after a rebuild, then restart the pilot. The
`placeholder` used for the adaptation check is not a Cloudflare credential and
is never used to serve traffic. A **separate** `homelab-cloudflare-dns-caddy`
token lives in Bitwarden; for this pilot it was copied without printing into
`~/.config/homelab/credentials/caddy-cloudflare-token` (directory `0700`, file
`0600`, both outside Git). The user unit's `LoadCredential=` reads that file;
`run-pilot.sh` passes it to Caddy as `CF_API_TOKEN`. The OpenTofu operator token
is not used. The credential file is not automatically refreshed when Bitwarden
rotates the token; replace it securely and restart the unit after rotation.

The installed user unit came from `caddy-pilot.service`. Stopping it makes both
custom-domain URLs unavailable while the TCP forward remains configured. To
inspect or stop it:

```sh
systemctl --user status caddy-pilot.service
systemctl --user stop caddy-pilot.service
```

To retest on the Framework without changing DNS, use `curl --noproxy '*'` with
`--resolve NAME:8444:127.0.0.1` and the corresponding `https://NAME:8444/`
URL. Do not use `-k`: certificate verification is part of the test. Ansible
can later install the binary and manage the unit, deploying this host's
file after validation. Other hosts get their own Caddyfile.

## Access review and cutover

The Tailscale admin policy was reviewed: its `{"action":"accept","src":["*"],
"dst":["*:*"]}` ACL allows all tailnet devices to reach Framework WSL on
TCP 443. This is the intended audience for both OpenCode and login-free HLIMS
staging; it does **not** grant access to devices outside the tailnet. A rule
permitting 443 applies to both hostnames, since the policy controls a port,
not Caddy's HTTP hostname. Public DNS A records point to the Tailscale
`100.x` address with Cloudflare proxying **off**; they don't make a public
route to that address. DNS names and issued certificate names are still public
(including in certificate-transparency logs). Leave Caddy on loopback, use
Tailscale Serve only, and **do not enable Funnel**. Revisit this decision if
untrusted users or devices join the tailnet or HLIMS begins storing more
sensitive data.

The pilot unit is enabled for user-session startup but `loginctl` reports
`Linger=no`: it will not run when this WSL user manager is stopped. That is
consistent with the existing on-demand user services; enabling linger would
also affect other enabled user units and is not part of this pilot. With WSL
and its user manager stopped, this ingress is unavailable.

The attended cutover removed the previous HTTPS 443 Serve handler with
`tailscale serve --https=443 off` (Tailscale does not permit HTTPS and raw TCP
on the same port), then installed the background tailnet-only route using
`tailscale serve --bg --tcp=443 tcp://127.0.0.1:8444`. The CLI's `--tcp=443`
selects raw TCP forwarding so Caddy, not Tailscale, terminates TLS. The former
Serve status was temporarily saved outside Git at
`/tmp/opencode/framework13-wsl-serve-before-caddy.json` (do not rely on this
temporary file for future rollback). Verify the current
route with `tailscale serve status --json` and curl both HTTPS hostnames from
another tailnet device with ordinary certificate verification (no `-k`). The
old `.ts.net` OpenCode HTTPS URL no longer works. Test the user unit and
forwarder after a WSL restart; their full restart behavior has not yet been
verified.

Rollback: `tailscale serve --tcp=443 off`, then
`tailscale serve --bg --https=443 http://127.0.0.1:4096` to restore the old
background HTTPS Serve proxy for `framework13-wsl.bullhead-tegus.ts.net`.
Confirm the old URL works again with verified TLS.
Do not reset other Serve configuration or delete DNS records as a substitute
for restoring the working route. The Framework's local user was granted
Tailscale operator privileges for the cutover and rollback. When re-enrolling
the node or changing its Tailscale IP, update the DNS plan and review it before
applying.
