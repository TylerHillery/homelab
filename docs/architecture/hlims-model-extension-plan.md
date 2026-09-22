# HLIMS Model Extension Plan

Status: proposed implementation plan for Clusters, Instance placement, physical
ports, network interfaces, prefixes, address pools, and cables.

## Goal

Close the model gaps required by the near-term homelab build without turning
HLIMS into a scheduler, router controller, firewall manager, monitoring system,
or general-purpose topology editor.

The implementation sequence is:

1. Clusters.
2. Cluster Instance placement.
3. Cluster topology and presentation.
4. Physical ports.
5. Machine network interfaces and Address ownership.
6. Network Prefixes.
7. Address Pools.
8. Stable Network selection.
9. Cables and port tracing.
10. Remaining presentation and demonstration coverage.

## Development Rules

The initial schema is unshipped and may be changed directly. Implement the
final design in `00001_inventory.sql`; do not introduce compatibility
`ALTER TABLE` migrations, backfills, deprecated duplicate columns, or dual-write
behavior for schemas that do not need to be preserved. Once a retained database
has applied migration 1, freeze it and use new numbered migrations thereafter.

Each implementation phase is OpenAPI-first:

1. Update `api/openapi.yaml`.
2. Regenerate API models, clients, and server interfaces.
3. Update `00001_inventory.sql` directly.
4. Add or update sqlc queries.
5. Regenerate sqlc output.
6. Implement handlers and domain validation.
7. Extend the API client and generic CLI.
8. Add schema, API, resolver, and client tests.
9. Update architecture and sample documentation.
10. Run generation checks, formatting, lint, SQL lint, ordinary tests, race
    tests, down-migration validation, and repository pre-commit hooks.

Preserve existing conventions:

- Internal IDs are application-generated UUIDv7 values stored as `TEXT` and
  are not normally exposed.
- Public IDs are application-generated, 12-character lowercase NanoIDs used by
  APIs and relationships.
- User-addressable named resources have editable names and stable lowercase
  slugs.
- Timestamps are integer Unix seconds, matching the current schema.
- Public CRUD follows the existing collection `GET`/`POST` and item
  `GET`/`PUT`/`DELETE` convention. Do not introduce `PATCH` for only these
  resources.
- Aggregate children are replaced transactionally when they do not need
  independent public identity.
- SQLite foreign keys, uniqueness constraints, checks, and triggers enforce
  structural invariants.
- API code performs semantic validation that SQLite cannot express reliably,
  including IP parsing, containment, and overlap checks.

## Clusters

Use a first-class Cluster rather than a generic Machine Group. A k3s cluster is
an operational system with scheduler-level placement semantics. Generic tags,
saved filters, and deployment cohorts must not become Instance placement
targets accidentally.

### Model

```text
clusters
  id
  public_id
  name
  slug
  kind
  notes
  created_at
  updated_at

cluster_memberships
  cluster_id
  machine_id

cluster_member_roles
  cluster_id
  machine_id
  role
```

`cluster_memberships` has a composite primary key of Cluster and Machine.
`cluster_member_roles` has a composite primary key of Cluster, Machine, and role
and a composite foreign key to the membership.

The model follows these rules:

- `kind` is a trimmed, nonempty open string such as `k3s`.
- Roles are trimmed, nonempty open strings such as `server` and `agent`.
- Membership is many-to-many; one Machine may participate in multiple
  Clusters.
- Clusters do not belong to one Machine Provider or Area. Cross-provider and
  cross-area membership is valid.
- Empty Clusters are valid for staged inventory.
- No k3s quorum guard is added. HLIMS records actual inventory and does not
  enforce scheduler availability.
- No k3s-specific role enum is embedded in the database.
- Cluster deletion cascades memberships and roles but is restricted while an
  Instance references the Cluster.
- Machine deletion cascades memberships but remains subject to the Machine's
  existing restrictions.

### API

Expose aggregate CRUD:

```text
GET    /api/v1/clusters
POST   /api/v1/clusters
GET    /api/v1/clusters/{publicId}
PUT    /api/v1/clusters/{publicId}
DELETE /api/v1/clusters/{publicId}
```

Cluster writes contain the complete member set and each member's complete role
set. Omitted arrays are invalid; explicit empty arrays are valid. Resolve and
validate every Machine and role before mutation, then replace Cluster metadata,
memberships, and roles in one transaction. Responses use deterministic member
and role ordering and never return null arrays.

The generic API client and CLI expose `clusters`. Separate member mutation
endpoints are unnecessary while membership has no independent lifecycle.

## Cluster Instance Placement

An Instance is placed on exactly one Machine or one Cluster.

### Schema

Change `instances.machine_id` to nullable, add nullable `cluster_id`, and enforce
the exclusive placement directly with a row check:

```text
(machine_id IS NOT NULL) + (cluster_id IS NOT NULL) = 1
```

Use `ON DELETE RESTRICT` for both placement foreign keys. Replace the current
Machine-only Instance name and slug indexes with four partial unique indexes:

- Service, Machine, and name for Machine-placed Instances.
- Service, Machine, and slug for Machine-placed Instances.
- Service, Cluster, and name for Cluster-placed Instances.
- Service, Cluster, and slug for Cluster-placed Instances.

### API

Use an explicit placement object rather than two optional sibling IDs:

```json
{
  "servicePublicId": "service1234",
  "placement": {
    "type": "cluster",
    "clusterPublicId": "cluster12345"
  },
  "name": "Production",
  "port": 3000
}
```

The supported placement types are initially `machine` and `cluster`.
Cluster-placed workloads appear once under the Cluster and are never copied
under each member Machine. Runtime pod placement is ephemeral scheduler state
and is not stored in HLIMS.

Placement changes that conflict with existing Endpoints are rejected. HLIMS
must not silently delete Endpoints while moving an Instance.

### Deferred Cluster Endpoints

Cluster Endpoints are deliberately deferred. Initially:

- Cluster-placed Instances cannot have Instance Endpoints.
- Endpoint insert and update validation rejects Cluster-placed Instances.
- No Cluster redirect path is introduced.
- No Cluster ad hoc port route is introduced.
- Cluster topology may display the workload without a resolver path.

This avoids treating an arbitrary member address as the stable identity of a
scheduled service. When a real ingress, load-balancer address, or VIP exists,
Cluster Address and Endpoint ownership will receive a separate design.

VIPs, anycast, and reverse-proxy ownership are outside this plan.

## Cluster Topology And Presentation

Clusters are top-level operational systems, not children of Provider or Area.
The topology projection becomes:

```text
Topology
|- Clusters
|    |- Members and roles
|    `- Cluster-placed Instances
`- Providers
     `- Areas
          `- Machines
               `- Machine-placed Instances
```

Cluster member records should reference concise canonical Machine summaries
rather than duplicate complete recursive Machine trees.

The Web Console gains a top-level Clusters section. Cluster details show member
roles and Cluster-placed Instances. The TUI remains read-only and may add the
same root-level Cluster navigation after the topology contract is stable.

## Physical Ports

Product port profiles remain grouped Product templates. They are not physical
ports, do not carry live connectivity, and cannot be Cable endpoints.

Add stable Asset-owned physical ports:

```text
physical_ports
  id
  public_id
  asset_id
  name
  connector
  speed_mbps
  position
  notes
```

Rules:

- Port name and position are unique within an Asset.
- Ports initially belong to router, switch, access-point, or network-adapter
  Assets.
- Product port profiles may assist port creation, but instantiated ports retain
  independent stable identities.
- Replacing or reordering Product port profiles cannot silently change physical
  port identity.
- Deleting a connected port is restricted until its Cable is disconnected.

Examples:

```text
I350 network-adapter Asset
  Port 0
  Port 1

Switch Asset
  Port 1
  Port 2
  ...
  Port 8
```

Expose ordinary top-level CRUD at `/api/v1/physical-ports` and
`/api/v1/physical-ports/{publicId}`.

## Logical Network Interfaces

Logical interfaces such as `eno1`, `wg0`, bridges, bonds, and VLAN interfaces
belong to Machines. They are distinct from Asset-owned physical ports.

```text
network_interfaces
  id
  public_id
  machine_id
  name
  kind
  mac_address
  mtu
  vlan_id
  parent_interface_id
  physical_port_id
  notes
```

Initial interface kinds are:

```text
physical
virtual
bridge
bond
vlan
tunnel
```

Rules:

- Interface name is unique per Machine.
- Parent and child interfaces belong to the same Machine.
- Direct and transitive parent cycles are rejected.
- VLAN interfaces require a parent and VLAN ID in the range 1 through 4094.
- Non-VLAN interfaces cannot carry a VLAN ID.
- Tunnel interfaces cannot carry VLAN IDs or physical-port bindings.
- A physical port binds to at most one logical interface.
- A physical binding is valid only when the port belongs to the Machine's
  backing system Asset or one of its descendant Assets.
- MAC addresses are normalized by the API but are not globally unique.
- Full trunk and access-port configuration is deferred.

Dogfood examples are:

```text
OPNsense Machine
  WAN -> I350 Port 0
  LAN -> I350 Port 1
  wg0 -> tunnel with no physical port

Mini Machine
  eno1 -> integrated or installed network-adapter port
```

Expose ordinary top-level CRUD at `/api/v1/network-interfaces` and
`/api/v1/network-interfaces/{publicId}`.

## Address Ownership

Because the initial schema is unshipped, replace the obsolete Machine Address
representation rather than preserving duplicate ownership and interface-name
columns.

An Address is owned by exactly one of:

- A Network Interface, deriving Machine ownership.
- An Area.
- An eligible network-equipment Asset.

Remove `addresses.machine_id` and `addresses.interface_name`. Keep direct Area
ownership for facts such as a public ISP address without a cataloged router.
Keep direct Asset ownership for router, switch, and access-point management
addresses.

Rules:

- Machine Addresses must reference a Machine-owned Network Interface.
- Area-owned and Asset-owned Addresses cannot reference an interface.
- A Machine-placed Instance Endpoint must reference an Address on an interface
  owned by the same Machine.
- `is_primary` means preferred for one owner within one Network.
- At most one primary Address exists per owner and Network.

Address API payloads accept `interfacePublicId`, `areaPublicId`, or
`assetPublicId` according to the exclusive ownership rule.

## Network Prefixes

Remove the single `networks.cidr` field and add child prefixes:

```text
network_prefixes
  id
  public_id
  network_id
  cidr
  notes
```

Rules:

- A Network may have multiple IPv4 and IPv6 prefixes.
- CIDRs must be canonical and masked.
- Duplicate prefixes within one Network are rejected.
- Address family and containment are validated in the API.
- Prefix overlap policy is explicit and tested.
- The schema does not use an IPv4-only GLOB check as semantic validation.

Expose ordinary top-level CRUD at `/api/v1/network-prefixes` and
`/api/v1/network-prefixes/{publicId}`.

## Address Pools

Pools and reservations are bounded allocation ranges within a Prefix, not
Prefixes themselves.

```text
address_pools
  id
  public_id
  network_prefix_id
  start_address
  end_address
  kind
  allocation_authority
  description
```

Initial pool kinds may include `dhcp`, `static`, `reserved`, and `vpn`.
`allocation_authority` records the authoritative allocator, such as manual,
OPNsense, or Omicron, without copying dynamic lease state.

Rules:

- Start and end share the parent Prefix's address family.
- Start is not greater than end.
- Both boundaries are contained by the parent Prefix.
- Overlapping pools under one Prefix are rejected unless a later concrete use
  case requires deliberately layered reservations.
- Dynamic DHCP leases and Omicron's internal allocation state are not copied
  into HLIMS.

Expose ordinary top-level CRUD at `/api/v1/address-pools` and
`/api/v1/address-pools/{publicId}`.

### Required OPNsense Input

The exact OPNsense DHCP start and end addresses are required before creating the
Lab LAN DHCP pool. Do not:

- Round the configured range to a convenient CIDR.
- Store `192.168.200.128/25` unless that is the actual configured range.
- Put an arbitrary start/end range into a CIDR column.
- Infer DHCP configuration from the desired Lab LAN Prefix.

If OPNsense is configured for `192.168.200.100` through
`192.168.200.199`, store those exact values in an Address Pool. The actual
values remain an implementation-time input and must not be guessed.

## Deferred Routing Features

Do not add the following proposed fields:

```text
networks.parent_network_id
networks.gateway_machine_id
networks.vlan_id
Network kind wireguard
NAT boolean or NAT uplink pair
```

The reasons are:

- `parent_network_id` conflates containment, routing, and overlay relationships.
- Gateways belong to routes, not intrinsically to Networks.
- NAT is traffic translation behavior, not a topology hierarchy or security
  policy.
- VLAN tags belong to interface/Network attachments and switching domains.
- WireGuard is an interface and tunnel technology, not necessarily a logical
  Network kind.

Future routed topology should use explicit typed relationships such as
`routed_to`, `overlay_on`, and a carefully defined `contains` relationship.
That work waits for a concrete consumer.

Firewall rules, NAT configuration, DHCP leases, WireGuard peers and keys,
Tailscale ACLs, and switch running configuration remain in their authoritative
systems.

## Stable Network Selection

Before multiple LANs become common, resolver selection must support stable
Network identity. Add selection by Network slug or public ID while retaining
the existing broad `via=lan|tailnet` behavior for compatibility.

Do not rely on `via=lan` once household, lab, and VLAN-backed LANs coexist.

## Cables

Cables are connectivity records, not Product or Asset inventory.

```text
cables
  id
  public_id
  name
  kind
  length_m
  color
  label_text
  endpoint_a_port_id
  endpoint_b_port_id
  notes
```

The endpoint columns are both null for a disconnected labeled Cable or both
non-null for an installed Cable.

Rules:

- One populated endpoint without the other is invalid.
- Endpoints must be different physical ports.
- A physical port appears in at most one installed Cable, regardless of which
  endpoint column contains it.
- Cable deletion disconnects and deletes the connectivity record.
- Physical port deletion is restricted while connected.
- Connector compatibility is validated only after connector vocabulary is
  sufficiently normalized.
- Patch panels, couplers, and media converters are Assets with physical ports,
  not multi-ended Cables.

Expose:

```text
GET    /api/v1/cables
POST   /api/v1/cables
GET    /api/v1/cables/{publicId}
PUT    /api/v1/cables/{publicId}
DELETE /api/v1/cables/{publicId}

GET /api/v1/physical-ports/{publicId}/connection
```

The port connection response includes the Cable and far-end port and Asset.

Initial real-world connections are recorded in label-maker order only after the
actual ports are confirmed:

```text
Deco LAN -> OPNsense WAN
OPNsense LAN -> switch Port 1
switch Port 2 -> mini-1
switch Port 3 -> mini-2
switch Port 4 -> mini-3
switch Port 5 -> Surface Book 3
```

These examples must be adjusted to match the installed wiring rather than
treated as authoritative seed data.

## Presentation

Presentation follows the domain instead of driving it:

- Web Console adds a top-level Clusters section.
- Cluster detail shows member roles and Cluster-placed Instances.
- Machine detail shows logical interfaces and physical-port bindings.
- Asset detail shows physical ports and Cable far ends.
- Network detail shows Prefixes and Pools.
- TUI remains read-only.
- CLI gains generic CRUD for every new top-level resource.
- A convenience Cable trace command may be added after the underlying API is
  stable.

The executable `--demo` data remains synthetic. It may gain synthetic examples
that exercise the new UI, but it must not become a hard-coded copy of private
live inventory. Real lab data is entered through ordinary APIs or a future
explicit import/bootstrap workflow.

## Topology Contract

`/api/v1/topology` remains a navigation projection, not complete inventory
topology. It may expose Clusters, members, roles, and Cluster-placed Instances
beside the existing Provider hierarchy.

Physical ports, interfaces, Prefixes, Pools, and Cables remain available through
their CRUD APIs. Homelable or another visualization consumer combines the
topology projection with those resources as needed.

Homelable remains a one-way projection consumer. This plan does not introduce
bidirectional synchronization or an HLIMS canvas model.

## Explicitly Deferred

- Firewall policy and rule configuration.
- NAT rule configuration.
- VIP and anycast ownership.
- Reverse-proxy ownership.
- Cluster endpoint resolution.
- Full 802.1Q trunk and access-port modeling.
- Complete routing tables and gateway modeling.
- WireGuard peer and key management.
- Dynamic DHCP leases.
- Runtime Kubernetes resources and health.
- Authentication and role-based authorization.
- Persisted operational health.
- Homelable synchronization beyond a one-way projection.

The wider ZTP, mise convergence, fnox/age, and OpenTelemetry/ClickHouse/Grafana
architectures remain separate projects and do not block this model work.

## Test Requirements

- Every new database constraint and trigger has positive and negative schema
  tests.
- Cluster aggregate replacement is atomic.
- Cross-provider Cluster membership is accepted.
- Duplicate memberships and roles are rejected.
- Machine/Cluster Instance placement exclusivity is tested.
- Cluster Instances reject Endpoints.
- Physical-port ownership and interface bindings are tested.
- Interface hierarchy cycles and cross-Machine parenting are rejected.
- VLAN and tunnel interface invariants are tested.
- Address ownership and Instance Endpoint ownership are tested.
- Prefix canonicalization, address family, containment, and overlap are tested.
- Pool boundary and overlap validation uses exact ranges.
- Cable zero-or-two endpoint rules are tested.
- One-Cable-per-port constraints are tested across both endpoint columns.
- OpenAPI round trips exist for every new resource.
- Generated code freshness remains part of `mise run check`.
- Goose up and down are tested against the final directly edited initial
  schema.

## Delivery Sequence

1. Add Cluster aggregate CRUD, memberships, and open roles.
2. Add exclusive Machine-or-Cluster Instance placement.
3. Add Cluster topology and presentation without Cluster redirects.
4. Add physical ports.
5. Add Machine Network Interfaces and rewrite Machine Address ownership.
6. Add Network Prefixes.
7. Obtain the actual OPNsense DHCP range, then add Address Pools.
8. Add stable Network resolver selection.
9. Add Cable connectivity and port tracing.
10. Add remaining Console, TUI, CLI convenience, documentation, and synthetic
    demo coverage.
