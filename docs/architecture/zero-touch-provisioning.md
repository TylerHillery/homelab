# Zero-Touch Provisioning

Status: planned project.

## Goal

After one-time firmware preparation, connecting an approved machine to power and
Ethernet should install its OS, join the tailnet, register inventory, converge
host configuration, and report ready without a local console.

## Workflow

```text
UEFI PXE -> iPXE -> approved profile -> unattended OS install
         -> single-use enrollment -> tailnet -> HLIMS
         -> machine convergence -> health checks -> ready
```

Existing appliances may use the same enrollment path through a temporary
management host. That host must not become a permanent dependency.

## Safety

- Unknown machines may report identity and hardware but may not modify disks.
- Match approved hardware by stable identifiers; add TPM identity later.
- Use expiring, single-use installation claims.
- Keep reusable credentials out of boot and cloud-init data.
- Mint one-off tagged Tailscale keys through a narrow enrollment identity.
- Verify direct management before removing the temporary path.
- Audit enrollment, retries, failure, and reprovisioning.

## Prerequisites

- Approved machine profiles and lifecycle states in HLIMS
- A narrow HLIMS enrollment operation
- An unattended secret backend with scoped machine identities
- A successful machine-management pilot
- A tested PXE, installer, and recovery network

## Initial Tooling

Use the smallest stack that satisfies the workflow: ProxyDHCP, iPXE, Ubuntu
autoinstall, cloud-init, a small enrollment broker, and the Tailscale API.
Consider MAAS, Foreman, Tinkerbell, or immutable operating systems only when
fleet size or reprovisioning frequency justifies them.

ZTP hands persistent convergence to [Machine Management](machine-management.md)
and follows the credential boundaries in
[Secret Management](secret-management.md).
