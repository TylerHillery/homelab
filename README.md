# Homelab

Monorepo for homelab software, infrastructure, and operations.

## Projects

- [`services/hlims`](services/hlims/): inventory, service catalog, resolver, and
  management clients
- [`opnsense`](opnsense/): declarative firewall, DHCP, DNS, and Tailscale
  configuration

## Repository

```text
docs/           architecture, decisions, and archived designs
mise-tasks/     shared executable tasks
opnsense/       OPNsense Ansible project and setup runbook
services/       applications and platform services
```

Add top-level directories only for active work. See
[`docs/architecture`](docs/architecture/) for current boundaries and planned
projects.

## Development

[mise](https://mise.jdx.dev/) manages tools and tasks.
[hk](https://hk.jdx.dev/) runs repository checks and Git hooks.

```sh
mise install --monorepo
mise exec -- hk install --mise
mise run check
mise run precommit
```

Root mise configuration owns shared tools and operations. Projects add local
tools and tasks in their own `mise.toml`. Use `mise //path:task` across projects
or `mise run task` inside a project.

Secrets are resolved through fnox and Bitwarden:

```sh
mise run bw:bootstrap-age
mise run bw:unlock
mise run fnox:exec <command>
```

`bw:bootstrap-age` creates the ignored local fnox configuration when needed.

Historical platform plans remain in [`docs/archive`](docs/archive/).
