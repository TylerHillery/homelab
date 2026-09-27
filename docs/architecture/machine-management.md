# Machine Management

Status: planned pilot for Ubuntu application hosts.

## Goal

Evaluate mise bootstrap as the host converger without replacing working Ansible
automation prematurely. Ansible remains the active OPNsense manager.

## Boundaries

| Mechanism | Responsibility |
|---|---|
| mise tools | Pin operator and build CLIs |
| mise environments | Supply non-secret configuration |
| mise tasks | Expose repeatable commands |
| mise bootstrap | Converge host packages, files, accounts, services, and workloads |
| fnox | Deliver secret values at execution time |
| HLIMS | Supply machine identity and inventory |

Tasks are commands, not automatically idempotent resources. Do not rebuild a
general configuration-management framework in shell. Use Ansible when a mature
module, fleet orchestration, or complex fact model is required.

## Pilot

Use one non-critical host:

1. Converge accounts, packages, SSH policy, Tailscale, and host firewall.
2. Converge one systemd service and one container workload.
3. Confirm repeated plans are empty and services do not restart unnecessarily.
4. Test interrupted convergence, secret rotation, and host recovery.
5. Record custom code and decide whether mise remains simpler than Ansible.

Unattended secret-bearing convergence depends on the machine identity model in
[Secret Management](secret-management.md). Remote execution must preserve SSH
host verification and must not forward broad operator credentials.
