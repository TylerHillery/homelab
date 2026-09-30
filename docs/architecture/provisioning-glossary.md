# Bare-Metal Provisioning: Mental Map & Glossary

> Terminology companion to [Machine Provisioning](provisioning-spec.md). This is a
> map of possible tools and stages, not a commitment to particular imaging,
> notification, or unattended-installation technology.

## The pipeline (the map)

Every bare-metal provisioning flow, from datacenter to homelab, is these stages.
Each stage has its own tools; each tool owns exactly one stage.

```
Power on
  │
  ▼
STAGE 0 ── Network boot ............ PXE / iPXE ────────────── ProxyDHCP (dnsmasq; OPNsense DHCP untouched)
  │         (how the machine loads   netboot.xyz (prebuilt menu)
  │          its first program)
  ▼
STAGE 1 ── Unattended OS install ... Ubuntu autoinstall ───── user-data (YAML answer file)
  │         (OS onto disk, no        kickstart (RHEL/Fedora/Rocky)
  │          keyboard)               preseed (old Debian; legacy)
  ▼
STAGE 2 ── First-boot identity ..... cloud-init ────────────── hostname, users, SSH keys
  │         (who am *I*?)            datasource: nocloud-net    phone_home module (!)
  ▼
STAGE 3 ── Configuration ........... Ansible ───────────────── roles = truth
            (day-0 tail + day 2..n)  (alternatives: Salt, Puppet/Chef, NixOS)

  OPTIONAL DAY-0 OPTIMIZATION, may replace part of the install path:
STAGE X ── Golden images ........... Packer (candidate) ────── image built BY Ansible roles
            (day-0 speedup)          (alternatives: mkosi, osbuild, KIWI)

  ORCHESTRATING stages 0–3 for you:
META ──── Provisioning managers .... Foreman / MAAS / Cobbler / Tinkerbell
          (prior art for a future     ← proposed provisioning service
          provisioning service)       lives here
```

Read it top to bottom and you can place any tool you've ever heard of.

---

## Stage 0 — Network boot

- **PXE** (Preboot Execution Environment): firmware-level standard. The NIC asks DHCP
  "who am I / where do I boot from?", then downloads a bootloader. Ancient, universal,
  limited (TFTP only, no scripting).
- **DHCP options 66/67**: common ways to specify a boot server and filename.
  The proposed lab flow instead uses ProxyDHCP to add boot information while
  OPNsense continues to own ordinary lab DHCP.
- **iPXE**: open-source network bootloader. Chainload it from PXE (or flash it)
  for HTTP boot, scripting (`menu`/`choose`), and per-device boot scripts. GRUB
  handles a different, usually local-disk boot stage.
- **TFTP vs HTTP boot**: TFTP is the legacy PXE transport (slow, fragile). iPXE lets
  the whole flow run over HTTP — kernels, initrds, and scripts served by any web
  server (or a future provisioning service).
- **netboot.xyz**: a prebuilt iPXE menu that boots dozens of OS installers straight
  from the internet. Useful shortcut if you ever want a boot menu *without* building
  a custom provisioning service first.

## Stage 1 — Unattended OS install

- **Ubuntu autoinstall**: since 20.04, Ubuntu's answer file system for the Subiquity
  (server) installer. You write a YAML config with an `autoinstall:` top-level section
  (disk layout, packages, user, which disk to wipe — the "don't nuke the wrong disk"
  part lives here), and the installer consumes it with zero keystrokes.
- **How the installer finds it**: kernel command line
  `autoinstall ds=nocloud-net;s=http://provisioning-host/` — i.e., "fetch my
  `user-data` over HTTP from this URL." A future service could generate per-device
  installer settings and identity; disk-selection policy needs explicit safeguards.
- **kickstart**: the Red Hat/Fedora/Rocky equivalent (older than autoinstall, `.cfg`
  files). Same idea, different distro family.
- **preseed**: Debian's older answer-file system (for debian-installer). Legacy on
  Ubuntu — autoinstall replaced it.

## Stage 2 — First-boot identity

- **cloud-init**: the industry-standard instance configurator. Its first-boot
  modules can set hostname, create users, install SSH keys, run scripts, and
  call back home. It commonly supplies a cloud VM's initial SSH key.
- **Datasource**: where cloud-init gets its config. Relevant ones: `NoCloud`
  (local files / ISO), **`nocloud-net`** (fetch `user-data` over HTTP), `MAAS`,
  `Ec2`. The `ds=` kernel parameter can select it.
- **`user-data` / `vendor-data`**: configuration documents cloud-init consumes.
  `user-data` is per-instance; `vendor-data` comes from the platform. A future
  service could generate per-device data after approval.
- **`phone_home` module**: cloud-init can POST instance data on first boot. A
  future service would still need to map that payload to its own workflow.

## Stage 3 — Configuration (day 0 tail, day 2 → ∞)

- **Ansible**: usually agentless over SSH; the existing OPNsense playbook runs
  locally against its API. It is the current and day-2 machine configuration
  tool. **Roles** are reusable units, **playbooks** compose roles for host groups,
  and **inventory groups** plus `group_vars/` supply group defaults.
- **Idempotency**: a correctly written playbook reports no changes when run again
  against its desired state. Verify this per role; it is not automatic for all
  tasks or a substitute for reviewing risky changes.
- **Convergence / drift**: convergence = repeated runs approach the defined state;
  drift = live state diverging from it (someone hand-edited a config). Ansible fixes
  drift by re-running; golden images can't.
- Alternatives in one breath: **SaltStack** (fast, agent-based), **Puppet/Chef**
  (agent-based, enterprise-legacy), **NixOS** (the radical alternative: the *entire
  system* is a pure function of a config file — no Ansible needed, but it's a
  lifestyle).

## Stage X — Golden images (optional day-0 optimization)

- **Packer** (HashiCorp, candidate): builds machine images from templates. A
  template declares **builders** (where to build: `qemu`, cloud APIs),
  **provisioners** (what to do inside: `shell`, `ansible`), and
  **post-processors**; the output is an **artifact** (the image file). The
  `ansible` provisioner could build an image *from* the machine roles.
- **Golden image**: a pre-built disk image intended to speed up day-0 setup. Its
  format and deployment method remain open; it should be a cache of role-defined
  state, not an independent source of truth.
- Alternatives: **mkosi** (systemd's own image builder — modern, declarative, worth a
  look when the time comes), **osbuild / Image Builder** (Red Hat family), **KIWI**
  (SUSE), **diskimage-builder** (OpenStack), **debos** (Debian).
- The footgun, restated: **never bake identity** (hostname, SSH host keys, tokens).
  Identity is injected at boot (cloud-init) and by Ansible (inventory), never imaged.

## Provisioning managers (prior art for a future service)

These orchestrate stages 0–3. A future homelab service might take inspiration
from them; its language and architecture are not chosen.

- **Foreman**: full lifecycle management — PXE, *discovery* (unknown machines
  PXE-boot a discovery image and register themselves — the industrialized version of
   a "new device wants to boot" flow), provisioning, and config-management
  integration (Puppet/Ansible plugins). Powerful, heavyweight, enterprise-y.
- **MAAS** (Canonical, "Metal as a Service"): API-driven bare metal with a cloud-like
  model — machines are *enlisted*, then *commissioned* (hardware inventory scripts
  run — this is the "commissioning" term), then *deployed*. Speaks to BMCs for power
  control. Ubuntu-centric.
- **Cobbler**: the elder — PXE + kickstart/preseed management, dead simple. Mostly
  legacy now. (Fun fact: written by Michael DeHaan, who later created Ansible.)
- **Tinkerbell** (CNCF sandbox, ex-Packet/Equinix Metal): the modern, cloud-native
  take — and philosophically the closest to what you want to build. It's *composable
  microservices*, everything behind APIs:
  - **Boots**: DHCP + iPXE serving (chainloading, `auto.ipxe`). A possible model
    for boot-file delivery.
  - **Hegel**: metadata service — machines query it for *their own* metadata.
  - **Rufio**: talks to BMCs (power, boot device). The minis do not have BMCs.
  - **Tink**: workflow engine — a controller schedules YAML workflows, workers on the
    target machines execute tasks.
  A useful design lesson: keep boot-file delivery, metadata, and workflow
  responsibilities separate even if a future implementation deploys as one service.

## Hardware layer terms (why your flow looks the way it does)

- **BMC** (baseboard management controller): a tiny computer on the motherboard that
  works when the main machine is off — power control, sensors, virtual console.
- **IPMI / Redfish**: protocols for talking to BMCs. IPMI is the old one, Redfish
  (REST-based) the modern one. Datacenter flows ("API: power on machine 7, boot from
  network") assume these.
- **The minis do not have BMCs.** An unattended flow must work without remote
  power or console control. A push notification is one possible approval channel,
  not a replacement for a BMC's power and recovery capabilities.

## The proposed pieces on the map

| Piece | Stage | Prior art |
|---|---|---|
| ProxyDHCP (dnsmasq) + iPXE | 0 | any PXE setup |
| iPXE menu + per-device scripts | 0 | Tinkerbell Boots, netboot.xyz |
| Provisioning service (approval, templated boot files; design TBD) | META | Tinkerbell Boots+Hegel, Foreman discovery |
| Optional push approval (transport TBD) | META | Foreman/MAAS web UI |
| Ubuntu autoinstall via per-device user-data | 1 | kickstart/preseed |
| cloud-init (`phone_home`, nocloud-net) | 2 | — (it *is* the standard) |
| Ansible roles/playbooks (current build) | 3 | — |
| Optional role-built golden images (builder TBD) | X | — |
| Later TPM attestation (design TBD) | META (trust) | Keylime and other attestation systems |

The current build is stage 3 (Ansible configuration). Images and automated
installation are separate later decisions; stage 3 remains the day-2 layer.

## The canonical config pattern ("one value, every consumer")

The rule: **give each setting one owner.** Ansible profile vars hold machine
settings it owns; mise may own packages on personal development devices with
dotfiles. Secrets and device-specific identity are separate runtime inputs.
Later consumers should reuse, rather than duplicate, relevant profile values:

- **Ansible** reads its own profile variables natively.
- **cloud-init** can use a rendered subset plus device-specific identity if an
  unattended installation flow is built.
- **An image builder**, if adopted, can run the roles without embedding identity.

The inputs fall into three categories:

- **Versioned inputs**: non-secret, profile-owned desired state such as DNS and
  sysctls. Hostnames may be versioned when appropriate, but not baked into a
  reusable image.
- **Runtime inputs**: private per-device details and credentials such as one-time
  Tailscale auth keys or cluster tokens; never store credentials in Git.
- **Build-time knobs** (optional): image variant flags and disk layout.

Change an Ansible-owned value → re-run the playbook to converge it. Services
restart only when the changed setting requires it; images need not be rebuilt
to apply a day-2 change.

## Attestation (the trust gate)

**The problem it could solve:** a future unattended flow may hand out secrets
(Tailscale auth keys, cluster tokens). A MAC address is spoofable. Attestation
could bind identity to hardware and measured boot state; the protocol is TBD.

**The vocabulary, one line each:**

- **TPM**: hardware chip (or firmware fTPM) that holds keys and measures each boot
  stage.
- **EK (Endorsement Key)**: unique per TPM, burned in by the manufacturer; the
  manufacturer's EK certificate proves "this is a genuine TPM."
- **PCR (Platform Configuration Registers)**: accumulate hashes of firmware →
  bootloader → kernel. A **quote** = PCR values signed by the AK. A tampered boot
  chain produces PCRs that differ from known-good.
- **AK (Attestation Key)**: TPM-generated; its certificate (chained to the EK)
  proves "this AK belongs to a genuine TPM" without exposing the EK itself.

Any gate before installation or secret delivery needs a boot-stage agent and a
verification protocol; neither has been selected. Attestation must be verified
before it can authorize a device, not merely included as an unchecked field in
a phone-home request.

**How it could compose with approval:** known, attested devices might qualify for
automatic approval; unknown devices could still require a human decision. Both
the approval channel and policy need design during the unattended phase.

**Practical notes:**

- Step zero: enable fTPM (AMD) / PTT (Intel) in the BIOS. Consumer boards ship with
  it off; verify TPM availability on each device before relying on it.
- The real operational cost isn't the crypto — it's **golden PCR management**. Every
  firmware update changes the known-good measurements, so you need a process for
  blessing new values. This is the part that bites in practice.
- **Keylime** is one build-vs-buy candidate. How a TPM quote would be collected,
  verified, and linked to enrollment is undecided.
- Sequencing: investigate attestation after automated provisioning. Operator-run
  Ansible does not require a new TPM workflow.
