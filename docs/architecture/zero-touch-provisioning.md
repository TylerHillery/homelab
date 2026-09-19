# Zero-Touch Provisioning

Status: planned as a separate ZTP project.

## Goal

After one-time firmware preparation, connecting a new mini PC to power and
Ethernet should install its OS, join Tailscale, register with HLIMS, and receive
its mise bootstrap configuration without a keyboard or console.

ZTP owns provisioning. HLIMS remains the inventory system of record, and mise
bootstrap owns ongoing Machine convergence as described in
[Machine Management](machine-management.md).

## Workflow

```text
UEFI PXE
  -> dnsmasq ProxyDHCP
  -> iPXE
  -> approved Machine profile
  -> pinned Ubuntu autoinstall
  -> cloud-init bootstrap
  -> one-time enrollment broker
  -> Tailscale bootstrap tag
  -> HLIMS registration
  -> mise bootstrap convergence
  -> ready
```

Unknown Machines may report hardware facts but must not modify disks until
approved. Pre-enrolled Machines should be matched using expected MAC, serial,
system UUID, and eventually TPM identity.

## Phases

### 1. Reliable Boot

- Prepare UEFI, PXE, boot order, and TPM settings once per Machine.
- Run dnsmasq as ProxyDHCP beside the existing router.
- Chainload a pinned iPXE binary and use HTTPS after the first boot stage.
- Keep netboot.xyz only as an explicit rescue option.

### 2. Discovery And Installation

- Add `expected`, `discovered`, `approved`, `installing`, and `failed` states.
- Collect hardware facts without changing disks on unknown Machines.
- Serve versioned Ubuntu autoinstall and cloud-init profiles.
- Select installation disks by stable serial or hardware path.

### 3. Secure Enrollment

- Give each installation an expiring, single-use claim.
- Keep reusable credentials out of iPXE, autoinstall, and cloud-init data.
- Let an enrollment broker mint a one-off tagged Tailscale auth key.
- Register observed hardware, OS, LAN, and Tailnet data through a narrow HLIMS
  enrollment operation.

### 4. mise Handoff

- Select the Machine's bootstrap roots from its approved profile.
- Connect over Tailscale and verify the Machine's SSH host identity.
- Apply host resources, deployment capabilities, and platform services.
- Mark the Machine ready only after health checks pass.

### 5. Hardening

- Sign and pin boot artifacts and the provisioning CA.
- Restrict the bootstrap Tailscale tag to enrollment and configuration traffic.
- Add audit, retry, recovery, and reprovisioning workflows.
- Evaluate Keylime TPM attestation after the basic pipeline is reliable.

## Provisioning Tools

| Tool | Intended Use |
|---|---|
| dnsmasq | ProxyDHCP and first-stage TFTP |
| iPXE | Hardware-aware HTTPS boot scripts |
| Ubuntu Subiquity | Deterministic unattended OS installation |
| cloud-init | Minimal first-boot bootstrap |
| ZTP enrollment broker | One-time claims and restricted credential exchange |
| Tailscale OAuth API | One-off, preauthorized, tagged Machine enrollment |
| HLIMS OpenAPI | Machine profiles, lifecycle state, and observed inventory |
| mise bootstrap | Persistent host and service convergence |
| Keylime | Optional later TPM attestation |

## Alternatives

| Platform | When To Consider It |
|---|---|
| MAAS | Frequent reprovisioning, larger fleets, or useful BMC support |
| Matchbox and Flatcar | Immutable container-host Machines |
| Foreman | Mixed enterprise operating systems and provisioning policies |
| Tinkerbell | Kubernetes-based bare-metal workflow orchestration |
| Cobbler | Traditional distro and PXE profile management |
| netboot.xyz | Interactive rescue and diagnostics only |

For a small x86 mini-PC fleet, dnsmasq, iPXE, Ubuntu autoinstall, and a small
HLIMS-aware enrollment broker provide the least duplicate inventory.

## Secret Management Dependency

Secret storage and credential policy are defined in
[Secret Management](secret-management.md). ZTP receives only the narrow
credentials needed to exchange a single-use installation claim for Tailscale
and HLIMS enrollment. Reusable credentials must not appear in iPXE,
autoinstall, or cloud-init data.
