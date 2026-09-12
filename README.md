# Homelab

Monorepo for the software and infrastructure that operate my homelab.

The first active project is [HLIMS](services/hlims/): the Home Lab Information
Management System. HLIMS is a private catalog for physical devices, machines,
services, running instances, endpoints, and memorable `go/*` links.

HLIMS is a system of record and navigation layer. It does not proxy application
traffic: catalog results and short links lead clients directly to service
instances over Tailscale.

## Layout

```text
docs/
  architecture/               Domain and system design
  decisions/                  Architecture decision records
  archive/                    Deferred ideas retained for later
services/
  hlims/                      HLIMS server, API, UI, and CLI
infra/
  environments/              Environment-specific desired state
  components/                Reusable infrastructure components, when needed
```

New top-level directories are added only for concrete work. Future services,
such as device attestation, will live beside HLIMS under `services/`.

## Development

[mise](https://mise.jdx.dev/) is the repository entrypoint for tools and tasks.
[prek](https://prek.j178.dev/) runs the required pre-commit checks.

```bash
mise install --monorepo
mise run check
mise run hooks:run
```

Each project owns its tools and tasks in its local `mise.toml`. From the
repository root, use paths such as `mise //services/hlims:test`; from anywhere
inside that project, use `mise run test`. Root tasks apply across the repository.

See [`docs/architecture/hlims.md`](docs/architecture/hlims.md) for the planned
application boundaries and implementation sequence.

The previous provisioning, attestation, and metrics plan remains in
[`docs/archive/initial-platform-plan.md`](docs/archive/initial-platform-plan.md)
and on branch `archive/initial-platform-plan-2026-09-12`.

## Hardware

| Brand | Model               | CPU                           | RAM | Storage   | Quantity |
| ----- | ------------------- | ----------------------------- | --- | --------- | -------- |
| HP    | ProDesk 600 G1 Mini | Intel Core i5-4590T @ 2.00GHz | 8GB | 256GB SSD | 3        |
