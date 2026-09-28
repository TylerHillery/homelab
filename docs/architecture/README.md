# Architecture

Architecture documents describe stable boundaries and active plans. Code,
OpenAPI, migrations, and runbooks remain authoritative for implementation and
operations.

| Document | Status | Scope |
|---|---|---|
| [HLIMS](hlims.md) | Current | Application boundaries and request flow |
| [HLIMS Domain Model](domain-model.md) | Current | Inventory concepts and relationships |
| [Home Network](home-network.md) | Current | Network topology and policy |
| [Secret Management](secret-management.md) | Current | Interactive secret storage and delivery |
| [HLIMS Roadmap](hlims-roadmap.md) | Planned | Model additions triggered by real needs |
| [HLIMS Console Pages](hlims-console.md) | Current / planned | Inventory, orders, and future DNS navigation |
| [Machine Management](machine-management.md) | Pilot | Host convergence evaluation |
| [Zero-Touch Provisioning](zero-touch-provisioning.md) | Planned | Unattended installation and enrollment |
| [Observability](observability.md) | Planned | Central telemetry pipeline |
| [Shared Machine Configuration](ansible-machine-profiles.md) | Proposal | Reusable platform-specific Tailscale automation |

Historical designs belong in [`docs/archive`](../archive/).
