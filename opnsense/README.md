# OPNsense

Ansible manages OPNsense through its HTTPS API. See the [setup
runbook](SETUP.md) for a fresh installation.

## Requirements

- Network access to the configured Tailscale hostname
- An ignored `group_vars/all/site.yml` based on `site.example.yml`
- API credentials in the `homelab-opnsense-api` Bitwarden item
- A Tailscale enrollment key in `tailscale-auth-key` when login is required
- An unlocked Bitwarden session

Bootstrap an operator device from the repository root:

```sh
mise install --monorepo
mise exec -- bw login
mise run bw:bootstrap-age
mise run bw:unlock
```

## Apply

From `opnsense/`:

```sh
mise run syntax
mise run check
mise run apply
```

Review `check` before applying. The management hostname is configured in
the ignored `group_vars/all/site.yml`; start from `site.example.yml`.

## Scope

The playbook manages:

- Tailscale enrollment, SSH, exit-node advertisement, and Serve
- Trusted tailnet HTTPS access
- Lab isolation and firewall aliases
- DHCP and Unbound DNS

Physical interface assignment, initial API credentials, and encrypted
configuration backups remain bootstrap and recovery tasks.
