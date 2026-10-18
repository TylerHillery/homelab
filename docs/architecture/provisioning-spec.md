# Machine Provisioning

Status: The shared Ansible project manages the Ubuntu development baseline on
Badger, Brewer and Buck. The user ran the full mise bootstrap separately.
Real second applies of the final clone-only dotfiles role are still pending
on all three; see the [rollout runbook](../../infrastructure/ansible/README.md).
Imaging, unattended installation, and attestation are later design directions,
not implementation commitments. See the [glossary](provisioning-glossary.md).

## Boundaries

- **Ansible converges machines**: accounts, system services, network configuration,
  drivers, and other profile-owned system state. It also remains the day-2 tool
  after a machine is installed. It does not install the operating system.
- **mise owns personal userland**: dotfiles, user tools, and bootstrap tasks. On
  personal development devices it may also own system packages defined in the
  dotfiles repo; Ansible may install that list from the checked-out config with
  normal privilege escalation, but must not duplicate the package declarations.
  Profiles without dotfiles can opt into Ansible-owned system packages. Assign
  each setting one source of truth rather than relying on a universal package rule.
- **HLIMS owns durable inventory**: machine identity, placement, relationships,
  addresses, and management endpoints. It does not store credentials, Tailscale
  policy, DHCP leases, or Ansible's desired configuration.
- **OPNsense owns lab DHCP and firewall policy.** A future network-boot service
  may supply boot information via ProxyDHCP without replacing OPNsense DHCP.
  OPNsense is an Ansible configuration target, not a netboot/OS-install target.
- **fnox and Bitwarden** supply secrets to interactive runs. Unattended automation
  needs a separate, narrowly scoped backend; see [Secret Management](secret-management.md).

## Current build: Ansible machine configuration

`infrastructure/ansible/` contains the `common`, `tailscale`, `dotfiles`, and
`podman` roles and the `ubuntu-dev.yml` composition. The separate opt-in Caddy
ingress role is not in that composition and does not manage Tailscale Serve.
Shared Cloudflare DNS is managed from `infrastructure/opentofu/cloudflare-dns/`,
not by the machine configuration.

The **intended layout** below includes both implemented and proposed roles and
playbooks. Entries marked `proposed` are not files to run today:

```text
infrastructure/ansible/
  ansible.cfg
  requirements.yml
  inventory/
    example.yml             # current: named hosts, no credentials
    group_vars/             # current: non-secret profile defaults
    host_vars/              # current: per-host exceptions
  roles/
    common/                 # current: Ubuntu baseline
    tailscale/              # current: already-enrolled Ubuntu nodes
    dotfiles/               # current: mise-owned development userland
    podman/                 # current: rootless containers
    caddy/                  # current: opt-in tailnet ingress
    docker/                 # proposed: define scope before implementing
    wsl/                    # proposed: WSL-only settings
    ubuntu_surface/         # proposed: Surface-specific Linux configuration
    ubuntu_nvidia_gpu/      # proposed: reusable NVIDIA driver configuration
    opnsense_firewall/      # current: OPNsense API and Tailscale SSH
  playbooks/
    ubuntu-dev.yml          # current: Ubuntu development mini PCs
    caddy-ingress.yml       # current: opt-in ingress, separate from ubuntu-dev
    ubuntu-wsl.yml          # proposed: Ubuntu dev plus WSL settings
    ubuntu-surface-gpu.yml  # proposed: Ubuntu dev plus Surface and NVIDIA roles
    opnsense.yml            # current: opt-in OPNsense firewall configuration
```

A separate WSL2 Ubuntu distro was used to test the common and Podman roles
before the mini-PC rollout, then removed; it did not test an independent
Tailscale node. The proposed WSL playbook would add WSL-specific settings to
the Ubuntu composition. Surface and NVIDIA roles remain independently
reusable; their combined playbook describes a machine class, not one device.

Playbooks compose capability roles; there is no profile inheritance mechanism.
Start with a `tailscale` role whose shared interface describes policy and whose
platform-specific tasks implement it. Extract shared task files only when there
is actual common implementation to reuse. OPNsense uses the existing local,
API-driven connection with fact gathering disabled; route it explicitly via
inventory/playbook context, **not** `ansible_os_family` (FreeBSD facts do not
identify OPNsense). Never run Linux package or service tasks on the firewall.
Unsupported platforms should fail clearly rather than silently skip requested
configuration. Whether/how to manage Tailscale on illumos is deferred until
there is an illumos target.

Inventory groups select targets and defaults; host variables hold exceptions.
Commit example inventory and non-secret desired settings. Keep real connection
details in ignored local files if they are private or machine-specific to an
operator; keep credentials out of inventory entirely. A hostname is not itself
a secret, but per-device identity must never be baked into a reusable image.
Secrets such as one-time Tailscale keys and cluster tokens are runtime inputs,
never checked into Git, exposed in Ansible output, or left in generated service
files. Verify SSH host identity on remote runs; do not forward broad operator
credentials to managed hosts. Cloud-init and image builders can later consume
the same profile-owned values without copying their definitions; identity and
build-only inputs stay separate.

### First scope and OPNsense

1. The Ubuntu mini-PC composition is deployed on Badger, Brewer, and Buck;
   the operator confirms reviewed real second applies reported no changes.
   Reboots and post-reboot checks are deferred separately. Mise remains the
   package owner for personal dev devices that use dotfiles.
2. OPNsense runs through the opt-in [firewall playbook](../../infrastructure/ansible/OPNSENSE.md),
   independent of Ubuntu roles. The migrated playbook passed syntax checks, a
   reviewed check-mode preview, and a real apply reporting zero changes.
   Read-only checks confirmed firewall isolation, DHCP, DNS, Tailscale Serve,
   and API/SSH access after apply; physical console recovery remains separate.
   Check mode is a preview, not a guarantee of safety or an exact prediction
   of every task.

Tailscale policy should document per-host node name, tag, accept-DNS, Tailscale
SSH, and explicit opt-ins for exit-node advertisement, subnet routes, and Serve.
The current OPNsense site **already enables** exit-node advertisement and trusted
GUI Serve; preserve that behavior during migration, while making it explicit in
the new configuration rather than a Linux default. Keep `os-tailscale`, API
management, SSH/HTTPS checks, and clearing the one-time auth key after enrollment.
Ubuntu should enroll only when needed and avoid restarting an already healthy
node on a no-op run. A new host needs initial management or console access before
the tailnet can be its normal management path. Review firewall check output and
retain a recovery path before any apply.

For the Ubuntu mini PCs, Tailscale SSH is the normal Ansible transport.
Today the operator manually installs and enrolls Tailscale from console or
another initial management path before Ansible can reach the node; the role
only converges already-enrolled machines. Automated first contact, enrollment,
and per-device identity belong to the later unattended boot flow below.

Disable the separate OpenSSH server only after verifying Tailscale SSH,
privilege escalation, and local-console recovery on each host. Bitwarden/fnox
can supply a scoped one-time enrollment key for an interactive run, but cannot
provide first contact to an unenrolled machine by itself.

## Fast feedback and verification

1. **Static, on every change:** [hk](../../hk.pkl) runs the shared Ansible
   `yamllint`, `ansible-lint`, and playbook syntax checks. Tool versions are
   pinned via mise and the Ansible project's `uv.lock`.
2. **Local converge loop:** for new roles, use a disposable Ubuntu VM or WSL
   distro to assert end state and a no-change second run. WSL distros on this
   machine share a network namespace and `tailscale0`; do not enroll another
   normal Tailscale daemon there. WSL cannot validate independent network or
   kernel/boot behavior; test SSH transport and sudo on the actual lab host.
3. **Role isolation where useful:** evaluate Molecule with Podman containers
   for roles whose behavior is meaningful in OCI containers; keep an instance up for
   repeated `converge`/`verify` runs. Add idempotence checks and Ansible `verify.yml`
   assertions for promised end state. Do not require a container scenario for
   roles that need a real init system, kernel, reboot, or OPNsense API.
4. **System-level tests when required:** evaluate Incus system containers for
   fuller Linux/systemd behavior, and Incus VMs for kernel or boot behavior.
   Use snapshots where helpful. A delegated Molecule scenario is one integration
   path; choose it after the first role
   demonstrates the need. Run reviewed `--check --diff` on real lab hosts before
   applying, then verify the resulting state and repeatability.

The netboot-to-ready E2E loop belongs to the later unattended-provisioning
phase, not a per-commit hook. Local test environments must not use live
enrollment keys or change production firewall state.

## Later phases (design direction)

1. **Golden images, if useful:** evaluate Packer or another image builder for
   faster day-0 setup. Build images from the same Ansible roles, omit hostname,
   host keys, and enrollment tokens, and re-run Ansible after boot. Image format,
   builder, and whether imaging precedes unattended installation are undecided.
2. **Automated zero-touch provisioning:** after a tested machine-management
   pilot, use network boot (e.g. ProxyDHCP + iPXE), an unattended Ubuntu installer
   and cloud-init for per-device identity, and an approval/enrollment service to
   trigger Ansible convergence. Notification transport (ntfy is only an example),
   service name, controller host, polling/retry behavior, and boot API are not
   chosen. Installer input must include enough controlled disk/install settings;
   it is not merely an identity-only document.
3. **Device attestation:** investigate TPM-backed identity and boot measurements
   as a later trust gate. Keylime versus a custom implementation and any quote
   schema are deferred until this phase; a MAC address alone is not proof of
   device identity. Evaluate trust before removing human approval from an
   unattended flow.

### Entry criteria and safety for unattended provisioning

- Define approved machine profiles and lifecycle states in HLIMS and a narrow
  enrollment operation. The boot/approval service owns temporary workflow state;
  HLIMS owns the durable record. Keep the service's approval state distinct from
  the Ansible playbook/inventory-group definition of a machine profile.
- Establish scoped unattended secrets, expiring single-use installation claims,
  one-off tagged Tailscale enrollment keys, audit trails, and a tested PXE,
  installer, and recovery network before handing out unattended credentials.
- Unknown devices may announce themselves but must not wipe disks or receive
  secrets. Approve against stable hardware identifiers; stronger hardware trust
  is a later attestation concern. Keep reusable credentials out of boot data.
- Existing appliances may enroll through a temporary management host. Verify
  direct management before removing it; do not make that host a permanent
  dependency. Audit failures, retries, and reprovisioning.

The next step is to verify Ubuntu idempotence on the real hosts, then migrate
working firewall automation without losing its management path.
