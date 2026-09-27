# HLIMS Domain Model

Status: current.

HLIMS records owned hardware, runnable machines, network addresses, services,
and reachable endpoints.

## Identity

- Internal UUIDv7 IDs define database relationships.
- Public NanoIDs identify API resources.
- Editable names are display values.
- Stable lowercase slugs identify named resources in routes and automation.
- External serial numbers, provider IDs, and system UUIDs remain attributes.

## Inventory

| Resource | Meaning |
|---|---|
| Manufacturer | Canonical hardware vendor |
| Product | Reusable hardware model and kind-specific specification |
| Asset | One owned or managed physical item |
| Purchase | One transaction associated with one or more Assets |
| Machine Provider | Environment responsible for Machines and Areas |
| Area | High-level physical or provider location |

Assets may be placed in an Area, contained by a system or rack Asset, or left
unplaced. Contained Assets inherit their effective Area. Product port profiles
describe grouped model capabilities, not physical connections.

## Compute

A Machine is a runnable bare-metal or virtual environment. Bare-metal Machines
may reference a system Asset. Virtual Machines may reference a parent Machine.
Machine capacity describes logical resources; physical components remain Asset
inventory.

A Machine User records a login name, not credentials or SSH trust. One preferred
user can guide clients when constructing SSH commands.

## Networking

A Network is an address scope such as LAN, tailnet, cloud network, or public
Internet. An Address belongs to one Network and one Machine, Area, or eligible
network-equipment Asset.

The current model inventories addresses, not complete routing or cabling. Ports,
interfaces, prefixes, pools, VLAN attachment, and routed relationships are added
only when a real consumer needs them. See the [HLIMS Roadmap](hlims-roadmap.md).

## Services

A Service is a conceptual application. An Instance places one Service on one
Machine and records its backend port. An Instance Endpoint records one
client-facing scheme, address, port, and path. The resolver redirects clients to
an endpoint and never proxies traffic.

## Core Relationships

```text
Manufacturer -> Product -> Asset -> Machine
Machine Provider -> Area -> Machine
Network -> Address -> Machine, Area, or network Asset
Service -> Instance -> Instance Endpoint -> Address
Purchase -> one or more Assets
```

The API contract and persistence constraints are authoritative:

- [`api/openapi.yaml`](../../services/hlims/api/openapi.yaml)
- [`db/migrations`](../../services/hlims/db/migrations)
