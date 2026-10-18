# Machine and firewall configuration

`ubuntu-dev.yml` configures already-enrolled Ubuntu mini PCs one host at a
time: Tailscale policy, an admin/security-update baseline, rootless Podman,
and a user-owned dotfiles checkout with mise tools. It does **not** enroll
Tailscale, install Ubuntu, manage OpenCode or Tailscale Serve, change the
login shell, or run the full dotfiles bootstrap. Caddy ingress is a separate
[opt-in playbook](../ingress/caddy/README.md). The OPNsense firewall uses a
[separate playbook](OPNSENSE.md), never the Ubuntu composition.

## Prepare a new mini PC

Install Ubuntu x86-64, create a non-root sudo user with subordinate UID/GID
ranges, and enroll Tailscale SSH from the console. Keep console recovery.
Add the host to `inventory/example.yml` and the `dev:*` choices in `mise.toml`.
Review `inventory/group_vars/ubuntu_dev.yml`: existing hosts advertise exit
nodes; a new host may need different values. Do not put credentials in inventory.

From this directory, verify access and preview before applying to **one** host:

```sh
tailscale ssh tyler@HOST 'id -un'
mise run dev:preflight HOST
mise run dev:preview HOST
mise run dev:apply HOST
```

`dev:preview` prompts for sudo and runs `--check --diff`; `dev:apply` prompts
for sudo and confirmation. Review the preview before applying. The first
preview cannot fully predict work that requires an as-yet-uncloned checkout.
Do not pass sudo passwords in arguments, logs or inventory. Ansible never
updates or resets an existing dotfiles checkout: normal edits and `git pull`
remain the user's responsibility. It reads the checkout's package list,
installs those packages via apt, then runs only mise's `dotfiles,tools` phases.
Run the full `mise bootstrap --yes` interactively on the host if desired.

After applying, verify and run a **real** second apply; it should report
`changed=0`:

```sh
mise run dev:verify-common HOST
tailscale ssh tyler@HOST 'podman info >/dev/null && echo Podman OK'
mise run dev:preview HOST
mise run dev:apply HOST
```

`dev:verify-common` checks sudo group membership, unattended-upgrades policy,
and apt timers; Podman and the dotfiles/tool state need separate verification.

## Current rollout and follow-ups

Badger, Brewer, and Buck have the baseline, user-run full dotfiles bootstrap,
editable `main` checkouts, and working rootless Podman. Read-only host checks
and static Ansible checks pass. The operator confirms reviewed real second
applies of the final clone-only dotfiles role on all three hosts reported no
changes. The Ubuntu mini-PC provisioning slice is complete.

Reboots and post-reboot persistence checks are deferred separately; they are
not a prerequisite for completing this provisioning slice. OpenCode user
units and Serve are outside this profile. Do not disable OpenSSH until the
Tailscale SSH and console recovery path is proven for that host. The role
supports only already-enrolled nodes and never changes Tailscale tags or
subnet routes implicitly. Sudo remains prompted; automated secret delivery,
new-host enrollment and WSL restart tests are follow-ups.

Run `mise //infrastructure/ansible:check` for yamllint, ansible-lint and
syntax checks. `inventory/local.yml`, if needed for connection details, is
ignored; no credentials belong there.
