# TODO

## OPNsense

Base configuration is in `opnsense/` and applied. A lab client has verified DNS,
public HTTPS, and upstream private-network isolation. Tailscale management is
enabled.

- [x] Add shared root mise tasks to provision the age identity from Bitwarden and refresh the age-encrypted `BW_SESSION` bootstrap.
- [x] Migrate OPNsense API credentials to Bitwarden references and verify retrieval; keep age only for the encrypted `BW_SESSION` bootstrap.
- [x] Block lab clients from the upstream private subnet while retaining DNS and public HTTPS access.
- [x] Join OPNsense to Tailscale as `tag:homelab` with a one-time pre-authorized key.
- [x] Allow direct HTTPS management from the tailnet with an Ansible-managed floating rule; no `tailscale0` assignment is required.
- [x] Configure Tailscale Serve for a browser-trusted OPNsense HTTPS endpoint.
- [ ] Track tailnet access policy as code. The four Tailscale-enabled servers need no subnet route.
- [ ] Select a durable Ansible runner that can reach OPNsense.

## Repository cleanup

- [x] Remove the leftover generated `ansible/` collection directory; keep OPNsense content under `opnsense/`.
- [x] Keep OPNsense docs short and use consistent, beginner-friendly terms.
- [ ] Review repository-wide dead files and stale ignores.
- [x] Add concise comments for floating-rule ordering and the Unbound prefix requirement.
- [x] Add Ansible syntax validation to root checks and pre-commit.

## Other work

- [ ] Add tests for down migrations to ensure they work.
- [ ] Investigate WSL/Tailscale DNS ownership warning and whether `tailscale set --accept-dns=false` is the appropriate fix.
- [ ] Create a separate ZTP project for PXE discovery, OS installation, repeatable Tailscale enrollment through temporary bootstrap paths, HLIMS registration, and mise bootstrap handoff.
- [x] Use Bitwarden Password Manager with fnox for interactive secret retrieval.
- [ ] Evaluate Bitwarden Secrets Manager for unattended automation.
- [ ] Pilot mise bootstrap as the Machine converger on one non-critical host before replacing Ansible.
- [ ] Evaluate an authenticated browser terminal backed by Tailscale SSH and ghostty-web, including session brokering, authorization, auditing, WASM asset delivery, and terminal lifecycle.
- [ ] Create a centralized observability project using host OpenTelemetry Collectors, a durable gateway, ClickHouse, Grafana, and wide structured events, with HyperDX and DuckLake evaluated as optional layers.
- [ ] Add VLANs, IPv6, or specialized address reservations only when a concrete workload requires them.
