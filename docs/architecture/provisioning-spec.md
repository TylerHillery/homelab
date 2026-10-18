# Machine Provisioning

Status: Ansible machine configuration is the current build; imaging, unattended
installation, and attestation are later phases. Their tools and detailed flows are
proposals, not implementation commitments. See the [glossary](provisioning-glossary.md)
for boot and attestation terminology.

## Boundaries

- **Ansible converges machines**: accounts, system services, network configuration,
  drivers, and other profile-owned system state. It also remains the day-2 tool
  after a machine is installed. It does not install the operating system.
- **mise owns personal userland**: dotfiles, user tools, and bootstrap tasks. On
  personal development devices it may also own system packages defined in the
  dotfiles repo; Ansible must not manage the same packages there. Profiles without
  dotfiles can opt into Ansible-managed system packages. Assign each setting one
  owner rather than relying on a universal package-ownership rule.
- **HLIMS owns durable inventory**: machine identity, placement, relationships,
  addresses, and management endpoints. It does not store credentials, Tailscale
  policy, DHCP leases, or Ansible's desired configuration.
- **OPNsense owns lab DHCP and firewall policy.** A future network-boot service
  may supply boot information via ProxyDHCP without replacing OPNsense DHCP.
  OPNsense is an Ansible configuration target, not a netboot/OS-install target.
- **fnox and Bitwarden** supply secrets to interactive runs. Unattended automation
  needs a separate, narrowly scoped backend; see [Secret Management](secret-management.md).

## Current build: Ansible machine configuration

The shared project will live in `infrastructure/ansible/`. Shared infrastructure
belongs under `infrastructure/`; application-owned infrastructure can live beside
its application, with independent OpenTofu state where needed. Shared Cloudflare
DNS has a separate scaffold at `infrastructure/opentofu/cloudflare-dns/`; its
provider is pinned and the two Framework WSL DNS records have been applied.
Machine configuration does not own its state or DNS records.

```text
infrastructure/ansible/
  ansible.cfg
  requirements.yml
  inventory/               # example inventory; real connection details as needed
  group_vars/              # non-secret profile defaults
  host_vars/               # per-host exceptions, if needed
  roles/
    common/
    tailscale/
    dotfiles/               # thin wrapper around the existing mise userland
    docker/
    wsl/                    # WSL-only settings, composed with Ubuntu roles
    ubuntu_surface/         # future: Surface-specific Linux configuration
    ubuntu_nvidia_gpu/      # future: reusable NVIDIA driver configuration
    opnsense_firewall/      # future migration from opnsense/
  playbooks/
    ubuntu-dev.yml          # first build: Ubuntu development machines
    ubuntu-wsl.yml          # pilot: Ubuntu dev composition plus WSL settings
    ubuntu-surface-gpu.yml  # later: Ubuntu dev plus Surface and NVIDIA roles
    opnsense.yml            # later: Tailscale plus firewall configuration
```

These are intended compositions, not implemented files. `ubuntu-dev` is the
first build for the Ubuntu mini PCs; pilot its applicable roles against a
**separate WSL2 Ubuntu distro** on the Framework 13 before applying to a mini.
The WSL playbook adds WSL configuration to the Ubuntu composition. Surface and
NVIDIA roles remain independently reusable; their combined playbook describes a
machine class, not a particular Surface Book.

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

### First scope and OPNsense migration

1. Build and pilot `ubuntu-dev` and the WSL-specific composition. Start with
   `common`, `tailscale`, and the optional `dotfiles` wrapper; add `docker` as
   applicable. Mise remains the package owner for personal dev devices that use
   dotfiles. Validate each supported role's actual behavior on the pilot before
   applying to a mini PC.
2. Preserve the working [`opnsense/`](../../opnsense/) project while adding a
   shared playbook. Initially reuse/wrap existing behavior; then move its tasks,
   collection requirements, site variables, mise commands, and setup runbook
   into the shared layout. Retire the old project only after operational parity.
3. Compare syntax, reviewed check-mode output, real apply results, management
   access, and a no-change second run. Confirm firewall, DHCP, DNS, Tailscale
   Serve, and API/SSH access before removing the old path. Check mode is a
   preview, not a guarantee of safety or an exact prediction of every task.

Tailscale policy should document per-host node name, tag, accept-DNS, Tailscale
SSH, and explicit opt-ins for exit-node advertisement, subnet routes, and Serve.
The current OPNsense site **already enables** exit-node advertisement and trusted
GUI Serve; preserve that behavior during migration, while making it explicit in
the new configuration rather than a Linux default. Keep `os-tailscale`, API
management, SSH/HTTPS checks, and clearing the one-time auth key after enrollment.
Ubuntu should enroll only when needed and avoid restarting an already healthy
node on a no-op run. A new host needs initial LAN/SSH or console access before
the tailnet can be its normal management path. Review firewall check output and
retain a recovery path before any apply.

## Fast feedback and verification

1. **Static, on every change:** extend the existing [hk](../../hk.pkl) hooks with
   `yamllint`, `ansible-lint`, and syntax checks for the new playbooks. Pin tools
   and collections through mise and Ansible requirements as in `opnsense/`.
2. **Local converge loop:** use the separate WSL2 distro to exercise Ubuntu
   configuration over the intended connection, assert end state, and verify a
   second run makes no changes. WSL shares a kernel with Windows: it cannot
   validate Surface drivers, kernel changes, or every service/network behavior.
3. **Role isolation where useful:** evaluate Molecule with a container scenario
   for roles whose behavior is meaningful in containers; keep an instance up for
   repeated `converge`/`verify` runs. Add idempotence checks and Ansible `verify.yml`
   assertions for promised end state. Do not require a container scenario for
   roles that need a real init system, kernel, reboot, or OPNsense API.
4. **System-level tests when required:** evaluate Incus system containers/VMs and
   snapshots for Linux roles needing systemd or kernel behavior. A delegated
   Molecule scenario is one integration path; choose it after the first role
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

The first implementation is deliberately smaller than this roadmap: converge
Ubuntu reliably, learn from the WSL pilot, then migrate the working firewall
automation without losing its management path.
