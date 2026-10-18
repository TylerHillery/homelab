# OPNsense

`playbooks/opnsense.yml` composes the OPNsense firewall role. It runs locally
on the controller against the firewall's tailnet HTTPS API and Tailscale SSH;
it does **not** gather FreeBSD facts or run Ubuntu roles on the firewall.
See the [setup runbook](OPNSENSE-SETUP.md) for a new firewall.

The site-specific values belong in the ignored
`inventory/host_vars/opnsense.yml` (start from `inventory/opnsense.example.yml`),
not in Git. API credentials and any one-time enrollment key still come from
Bitwarden through the root `fnox:exec` task. Never pass secrets as playbook
arguments or include them in review output.

From the repository root:

```sh
mise run bw:unlock
mise //infrastructure/ansible:check
mise //infrastructure/ansible:opnsense:preview
```

**Before the first apply**, obtain an encrypted OPNsense configuration backup
outside Git, verify direct management and console recovery, and inspect the
preview for unexpected firewall, DNS, DHCP, or Serve changes. Then, only after
approval, run `mise //infrastructure/ansible:opnsense:apply`. Verify GUI/API,
DNS, DHCP, firewall isolation, Tailscale SSH/Serve, and exit-node access; a
second real apply should report zero changes.

API tasks mask their output with `no_log` to avoid exposing credentials;
the preview recap shows whether they would change, but not their diffs. Review
the role's declared rules alongside the recap and investigate any unexpected
changes before applying.

## Verified on 2026-10-18

The migrated real apply reported `changed=0`. Read-only checks confirmed:

- Tailnet GUI HTTPS returned 200 with valid TLS; authenticated API reads worked.
- Tailscale SSH reached OPNsense, Serve proxied TCP 443 to the local GUI,
  Tailscale was running, and the firewall was listed as an exit-node option.
- The live PF table contained the RFC1918 ranges; the API reported the
  management, lab DNS/GUI, and isolation rules enabled. From Badger, traffic to
  the upstream gateway was blocked while public HTTPS returned 200.
- The API's DHCP subnet, pool, gateway and DNS settings matched the site
  inventory; Badger, Brewer and Buck had DHCP-sourced lab routes.
- Badger resolved public DNS directly through the firewall; the required
  Unbound private domain was selected.

Physical console recovery and restoration from the encrypted backup were not
tested remotely.
