# Architecture

Architecture documents describe stable boundaries and active plans. Code,
OpenAPI, migrations, and runbooks remain authoritative for implementation and
operations.

| Document | Status | Scope |
|---|---|---|
| [HLIMS](hlims.md) | Current | Application boundaries and request flow |
| [HLIMS Domain Model](domain-model.md) | Current | Inventory concepts and relationships |
| [Home Network](home-network.md) | Current | Network topology and policy |
| [Infrastructure as Code Layout](infrastructure-as-code.md) | Current | OpenTofu root and state boundaries |
| [Secret Management](secret-management.md) | Current | Interactive secret storage and delivery |
| [HLIMS Roadmap](hlims-roadmap.md) | Planned | Model additions triggered by real needs |
| [HLIMS Console Pages](hlims-console.md) | Current / planned | Inventory, orders, and future DNS navigation |
| [Machine Provisioning](provisioning-spec.md) | Current build / later phases | Ansible host configuration, local testing, imaging, and future unattended installation |
| [Provisioning Glossary](provisioning-glossary.md) | Reference | Boot, installer, and attestation terminology |
| [Observability](observability.md) | Planned | Central telemetry pipeline |

Historical designs belong in [`docs/archive`](../archive/).
