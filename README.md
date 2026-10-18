# Homelab

Monorepo for homelab software, infrastructure, and operations.

## Projects

- [`services/hlims`](services/hlims/): inventory, service catalog, resolver, and
  management clients
- [`infrastructure/ansible`](infrastructure/ansible/): Ubuntu machine configuration
  and opt-in OPNsense firewall management
- [`infrastructure/ingress`](infrastructure/ingress/): tailnet-only HTTPS ingress
- [`infrastructure/opentofu`](infrastructure/opentofu/): shared DNS records and
  state storage bootstrap

## Repository

```text
docs/           architecture and archived designs
infrastructure/ shared machine, ingress, and DNS configuration
mise-tasks/     shared executable tasks
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

See [Infrastructure as Code Layout](docs/architecture/infrastructure-as-code.md)
for how OpenTofu roots, state, and reusable modules are organized.

Secrets are resolved through fnox and Bitwarden:

```sh
mise run bw:bootstrap-age
mise run bw:unlock
mise run fnox:exec <command>
```

`bw:bootstrap-age` creates `~/.config/homelab/fnox.local.toml` outside the checkout.

Historical platform plans remain in [`docs/archive`](docs/archive/).
