# Deferred HLIMS Cluster Design

This document preserves the Cluster model prototyped in September 2026. It is
not part of the current HLIMS domain or API. The implementation was removed
because the inventory does not yet contain real Clusters.

## Definition

A Cluster is a set of Machines cooperating as one operational system. It is a
top-level resource, independent of the physical Machine Provider and Area
hierarchy. This is intentionally different from an arbitrary Machine group,
tag, or saved filter.

Examples include Kubernetes, k3s, Consul, Nomad, Docker Swarm, Proxmox, and ECS.
A Cluster may span Machine Providers, and an empty Cluster is valid for managed
systems such as ECS Fargate or for inventory created before its members exist.

Cluster data is descriptive inventory. It does not schedule workloads, elect
control planes, converge configuration, or imply runtime health.

## Proposed Data Model

```sql
create table clusters (
    id         text    not null primary key,
    public_id  text    not null unique,
    name       text    not null collate nocase unique,
    slug       text    not null collate nocase unique,
    kind       text    not null,
    notes      text,
    created_at integer not null,
    updated_at integer not null
);

create table cluster_memberships (
    cluster_id text not null references clusters (id) on delete cascade,
    machine_id text not null references machines (id) on delete cascade,
    primary key (cluster_id, machine_id)
);

create table cluster_member_roles (
    cluster_id text not null,
    machine_id text not null,
    role       text not null,
    primary key (cluster_id, machine_id, role),
    foreign key (cluster_id, machine_id)
        references cluster_memberships (cluster_id, machine_id) on delete cascade,
    check (role != '' and role = trim(role))
);
```

Both `kind` and membership roles are trimmed, open strings rather than enums.
One membership can have zero or more roles. This supports combinations such as
`control-plane` plus `worker` without embedding Kubernetes-specific rules in
HLIMS.

## Proposed API

Expose aggregate CRUD at `/api/v1/clusters` and
`/api/v1/clusters/{publicId}`:

```json
{
  "name": "Homelab Kubernetes",
  "kind": "kubernetes",
  "members": [
    {
      "machinePublicId": "badmach8k2q5",
      "roles": ["control-plane", "worker"]
    },
    {
      "machinePublicId": "brewmach4n8p",
      "roles": ["worker"]
    }
  ]
}
```

Create and update requests must include the complete `members` array and every
member's complete `roles` array. Explicit empty arrays are valid; omitted arrays
are rejected. Resolve and validate all Machines and roles before changing data,
then replace metadata, memberships, and roles in one transaction. Reject
duplicate Machines and duplicate, empty, or padded roles. Return deterministic
member and role ordering with non-null arrays.

## Proposed Topology And Clients

The topology response should contain top-level `clusters` beside `providers`:

```text
Topology
|- Clusters -> Members -> Machine summaries and roles
`- Providers -> Areas -> Machines
```

Cluster member summaries resolve to canonical Machines globally rather than
within one Provider. The Web Console should show a separate collapsible Cluster
section with role badges. The TUI should show root-level Cluster nodes before
Providers and use canonical Machine details when a member is selected. The CLI
resource should be `hlims clusters`.

## Previous Prototype Map

The removed prototype used these integration points:

- `db/migrations/00001_inventory.sql`: the three normalized tables and cascades.
- `db/queries/clusters.sql`: aggregate CRUD support and topology rows.
- `api/openapi.yaml`: Cluster CRUD schemas and top-level topology models.
- `api_clusters.go`: required-array decoding, validation, and transactional
  complete-set replacement.
- `api_topology.go`: role-row aggregation and cross-provider member resolution.
- `internal/apiclient/client.go`: generic `clusters` resource dispatch.
- `internal/cli/root.go`: nested member/role create example.
- `internal/tui/model.go`: root Cluster nodes and global Machine lookup.
- `console/templates/topology.html`: top-level Cluster cards and role badges.
- `console/static/console.css`: Cluster section, cards, member chips, and roles.
- Schema, CRUD, topology, demo, Console, API client, CLI, and TUI tests covered
  cross-provider membership, empty Clusters, role constraints, ordering,
  cascades, and atomic replacement.

After restoring those inputs, run `mise run generate` from `services/hlims` to
regenerate sqlc and OpenAPI output. Then validate with `mise run check`, race
tests, static builds, Goose up/down, repository pre-commit hooks, and a live demo
topology request.
