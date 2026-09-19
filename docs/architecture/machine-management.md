# Machine Management

Status: pilot mise bootstrap as the primary Machine converger; retain Ansible
as a fallback until the pilot covers a complete host.

## Decision

Use mise bootstrap for the initial Ubuntu mini-PC fleet. Current mise releases
provide declarative resources for the work this homelab needs immediately:

- Linux users and groups.
- apt packages, including exact version pins.
- Privileged files and directories with ownership, modes, templates, and
  atomic replacement.
- Existing systemd system services and managed systemd user units.
- File-triggered service reloads and restarts.
- nftables, firewalld, or UFW policy with SSH lockout protection.
- Docker Compose lifecycle, config drift, health checks, and dependencies.
- Git repositories, tools, dotfiles, plans, status, and dry runs.
- Remote convergence over OpenSSH.

This is sufficient to test mise in place of Ansible without introducing a
second configuration language. It also aligns Machine convergence with the
repository's existing mise tasks and the fnox secret boundary.

## Configuration Layers

Use each mise feature for its intended responsibility:

| Feature | Responsibility |
|---|---|
| `[tools]` | Pin and install CLIs used by operators, CI, and bootstrap tasks |
| Environments | Supply non-secret per-project and per-environment configuration |
| Tasks | Provide named build, test, deployment, and operational entry points |
| Bootstrap resources | Inspect and converge Machine packages, files, accounts, services, firewall, Compose projects, repositories, and dotfiles |

This keeps one discoverable configuration surface without pretending all four
layers have the same convergence semantics. In particular, tasks are commands;
they are not automatically idempotent resources.

## Composition

Model reusable capabilities as independent configuration roots and select them
from a Machine configuration:

```toml
[bootstrap]
config_roots = ["../base", "../docker", "../wikijs"]
```

Selected roots compose dotfiles, managed files and directories, services, and
Compose projects. Identical declarations are deduplicated. Conflicting
declarations for the same resource fail with both origins instead of relying on
array order.

The composition model intentionally does not aggregate tasks, tools, packages,
hooks, or repositories. Keep those in the Machine's normal config hierarchy
until mise defines ordering and conflict semantics for them.

## Execution

Use plans before applying locally or remotely:

```text
mise bootstrap plan
fnox exec -- mise bootstrap --dry-run
fnox exec -- mise bootstrap
mise bootstrap remote <machine> --dry-run
mise bootstrap remote <machine>
```

Remote bootstrap stages the reviewed project and a compatible mise binary,
uses the normal OpenSSH host-key policy, runs convergence on the target, and
removes staging afterward. HLIMS remains the Machine inventory; a small adapter
or generated mise remote inventory can map HLIMS Machines to SSH targets when
the fleet is large enough to justify it.

A Machine can also start directly from a reviewed Git repository:

```text
mise bootstrap --from <configuration-repository> --yes
```

Use this after ZTP only when the Machine has narrowly scoped read access to the
repository. Otherwise, initiate `mise bootstrap remote` from an operator or CI
host and rely on its established SSH and repository authentication.

## Boundaries

Declarative mise resources inspect current state and skip unchanged resources.
Hooks and `[tasks.bootstrap]` run on every selected apply and must be written as
repeatable imperative operations. Bootstrap is a sequence, not a transaction;
earlier successful changes remain when a later phase fails.

Compared with Ansible, mise currently has a smaller resource and integration
ecosystem. Configuration such as mounts, sysctl values, authorized keys, or
specialized service APIs may need managed files or custom tasks. It also lacks
Ansible's mature dynamic inventory, fact model, rolling orchestration, and
broad collection ecosystem.

Do not recreate a general configuration-management framework in shell tasks.
If the pilot accumulates substantial custom convergence code, cannot safely
model host differences, or needs coordinated fleet rollouts, use Ansible for
those responsibilities instead.

## Secret Boundary

mise declares required secret inputs, while fnox supplies their values:

```text
fnox exec -- mise bootstrap
```

Local environment values are not forwarded by `mise bootstrap remote`.
Attended runs may use `--prompt-secrets`. For unattended secret templates, ZTP
must install fnox, persistent reviewed configuration, and a scoped age identity
so the Machine can run `fnox exec -- mise bootstrap` locally. Do not assume a
local fnox wrapper crosses the SSH boundary. See
[Secret Management](secret-management.md).

## Pilot

Prove the approach on one non-critical Machine before replacing Ansible:

1. Bootstrap accounts, base packages, SSH policy, Tailscale, and host firewall.
2. Install Docker and converge one Compose service with a health check.
3. Install and converge one systemd service with file-change notification.
4. Re-run until the plan is empty and no service restarts unnecessarily.
5. Exercise a failed mid-run apply, secret rotation, and host recovery.
6. Record every custom task needed and decide whether the remaining gap is
   smaller than operating Ansible alongside mise.
