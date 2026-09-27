# HLIMS Roadmap

Status: planned additions triggered by real consumers.

HLIMS should grow from observed workflows, not speculative topology detail.

| Need | Add when required |
|---|---|
| Multiple LANs need deterministic resolution | Stable Network selection by slug or public ID |
| One Network needs multiple address ranges | Network Prefix resources |
| Reserved or delegated ranges must be tracked | Address Pools with an allocation authority |
| Machine addresses need stable attachment | Logical Network Interfaces |
| Installed links need tracing | Asset-owned physical ports and Cables |
| Routed relationships need inventory | Explicit typed Network relationships |
| A real scheduler cluster exists | Cluster inventory and placement design |
| Proxy, VIP, or load balancer owns an endpoint | Explicit endpoint ownership model |

## Rules

- OpenAPI defines the contract before handlers and clients.
- Migrations preserve any database that has become durable.
- SQLite enforces structural invariants; API code enforces semantic networking
  rules.
- Runtime systems remain authoritative for routes, leases, firewall rules,
  scheduler state, and tailnet policy.
- Synthetic demo data belongs in executable fixtures, not architecture tables.
- Add presentation only after the underlying model and API are stable.

Cluster concepts remain in the archived
[Cluster Design](../archive/hlims-cluster-design.md).
