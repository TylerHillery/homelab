# HLIMS Architecture

Status: current.

HLIMS is an inventory-backed redirect resolver. It catalogs infrastructure and
returns client-reachable service URLs. It does not provision machines, schedule
workloads, monitor health, or proxy application traffic.

## Principles

- OpenAPI is the external contract and generates server and client code.
- Browser redirects and API resolution use the same resolver.
- Inventory relationships produce destinations; no separate link store exists.
- Normal database access uses sqlc-generated queries.
- Clients connect directly to the selected endpoint.
- Private ingress is a deployment concern, not an application dependency.

## Request Flow

```text
browser or API client
        |
        v
     resolver
        |
        v
Machine -> Service -> Instance -> preferred Instance Endpoint
        |
        v
HTTP redirect or JSON destination
```

Canonical routes select a Machine, Service, and Instance. Ad hoc routes select
a Machine and port. Network preference can distinguish LAN and tailnet paths.

## Components

| Component | Responsibility |
|---|---|
| `hlimsd` | HTTP API, resolver, redirects, and web console |
| `hlims` | API-only CLI and terminal console |
| OpenAPI | External request and response contract |
| SQLite migrations | Persistence constraints |
| sqlc | Application queries |

Generated code is committed, never edited manually, and checked by mise.

The web and terminal consoles consume a topology projection rather than joining
flat resources independently. Browser reachability indicators are best-effort
client observations and are not persisted as monitoring data.

## Responsibility Boundaries

| Layer | Responsibility |
|---|---|
| Service manager or container runtime | Run processes on a Machine |
| Scheduler or deployment controller | Place and release workloads |
| Machine converger | Configure hosts and services |
| Ingress publisher | Expose a backend at a stable address |
| HLIMS | Record identity, placement, relationships, and endpoints |

HLIMS records deployment results without replacing the system that produced
them.

## Deployment And Security

HLIMS serves HTTP on a configurable address. Deploy it behind trusted private
ingress such as Tailscale Serve or another authenticated network. Public ingress
is not required.

The current personal deployment has no application authentication. Any client
that can reach HLIMS can mutate inventory. Network policy is therefore the
security boundary. The server does not grant cross-origin browser access.

## Sources Of Truth

- [`services/hlims/api/openapi.yaml`](../../services/hlims/api/openapi.yaml): API
- [`services/hlims/db/migrations`](../../services/hlims/db/migrations): schema
- [`services/hlims/README.md`](../../services/hlims/README.md): operation and use
- [HLIMS Domain Model](domain-model.md): conceptual relationships
